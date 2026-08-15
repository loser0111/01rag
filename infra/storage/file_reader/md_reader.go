package file_reader

import (
	"bufio"
	"com.wyq.01rag/domain/model/document"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MarkdownFileReader 按 Markdown 结构切分 chunk 的文件读取器
type MarkdownFileReader struct {
	// ChunkMaxSize 单个 chunk 的最大字符数（软上限，尽量在段落边界切分）
	ChunkMaxSize int
}

func NewMarkdownFileReader(chunkMaxSize int) *MarkdownFileReader {
	if chunkMaxSize <= 0 {
		chunkMaxSize = 1000
	}
	return &MarkdownFileReader{ChunkMaxSize: chunkMaxSize}
}

// Read 实现 document.IFileReader 接口，读取 md 文件并切分为 chunks
func (r *MarkdownFileReader) Read(file *document.File) ([]*document.Chunk, error) {
	if file == nil {
		return nil, fmt.Errorf("file is nil")
	}
	path := file.FilePath
	if path == "" {
		path = file.FileName
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".markdown" {
		return nil, fmt.Errorf("unsupported file extension %s, only .md/.markdown supported", ext)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", path, err)
	}
	defer f.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return r.splitChunks(lines, file), nil
}

// headingLevel 返回 # 的层级（0 表示不是标题）
func headingLevel(line string) int {
	if !strings.HasPrefix(line, "#") {
		return 0
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i > 6 {
		return 0
	}
	if i < len(line) && line[i] != ' ' {
		return 0
	}
	return i
}

type mdSection struct {
	heading *string // nil 表示文件开头未标题部分
	level   int
	body    []string
}

// splitChunks 按标题/段落结构切分，超长时按句子/段落再细拆
func (r *MarkdownFileReader) splitChunks(lines []string, file *document.File) []*document.Chunk {
	sections := r.extractSections(lines)
	var chunks []*document.Chunk
	var pending strings.Builder

	flushPending := func() {
		text := strings.TrimSpace(pending.String())
		if text != "" {
			chunks = append(chunks, &document.Chunk{Data: text})
		}
		pending.Reset()
	}

	for _, sec := range sections {
		var sb strings.Builder
		if sec.heading != nil {
			sb.WriteString(*sec.heading)
			sb.WriteString("\n")
		}
		for _, line := range sec.body {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sectionText := sb.String()

		// 整个 section 就能塞下，直接作为一个 chunk，同时带上父级标题上下文
		if pending.Len()+len(sectionText) <= r.ChunkMaxSize {
			pending.WriteString(sectionText)
			continue
		}
		flushPending()

		// 单独 section 都超了上限，按段落再细拆
		if len(sectionText) > r.ChunkMaxSize {
			for _, subChunk := range r.splitLargeSection(sectionText) {
				chunks = append(chunks, &document.Chunk{Data: subChunk})
			}
		} else {
			pending.WriteString(sectionText)
		}
	}
	flushPending()

	// 所有 chunk 附上文档名，便于调试；Chunk 结构体只暴露 Data，故放在数据头部作为标识
	if file != nil && file.FileName != "" {
		header := fmt.Sprintf("[source: %s]\n", file.FileName)
		for i := range chunks {
			chunks[i].Data = header + chunks[i].Data
		}
	}
	return chunks
}

// extractSections 按 # 标题切分为若干 section
func (r *MarkdownFileReader) extractSections(lines []string) []mdSection {
	sections := make([]mdSection, 0)
	current := mdSection{level: 0}

	flush := func() {
		if current.heading != nil || len(current.body) > 0 {
			sections = append(sections, current)
		}
	}

	for _, line := range lines {
		lv := headingLevel(line)
		if lv > 0 {
			flush()
			h := line
			current = mdSection{heading: &h, level: lv}
			continue
		}
		// 跳过代码块，避免代码块里的 # 被误识别为标题
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			current.body = append(current.body, line)
			continue
		}
		current.body = append(current.body, line)
	}
	flush()
	return sections
}

// splitLargeSection 按段落/句子进一步拆分过大的文本
func (r *MarkdownFileReader) splitLargeSection(text string) []string {
	paras := strings.Split(text, "\n\n")
	var result []string
	var sb strings.Builder
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if sb.Len()+len(p)+2 <= r.ChunkMaxSize {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(p)
			continue
		}
		if sb.Len() > 0 {
			result = append(result, strings.TrimSpace(sb.String()))
			sb.Reset()
		}
		// 单个段落依然过大，按句子（。或 \n）再拆
		if len(p) > r.ChunkMaxSize {
			for _, s := range splitBySentence(p, r.ChunkMaxSize) {
				result = append(result, s)
			}
		} else {
			sb.WriteString(p)
		}
	}
	if sb.Len() > 0 {
		result = append(result, strings.TrimSpace(sb.String()))
	}
	return result
}

// splitBySentence 按中文句号/换行切分，达到上限就合并
func splitBySentence(text string, maxSize int) []string {
	separators := []rune{'。', '！', '？', '.', '!', '?', '\n'}
	var out []string
	var sb strings.Builder
	for _, ch := range text {
		sb.WriteRune(ch)
		isSep := false
		for _, s := range separators {
			if ch == s {
				isSep = true
				break
			}
		}
		if isSep && sb.Len() >= maxSize/2 {
			out = append(out, strings.TrimSpace(sb.String()))
			sb.Reset()
		} else if sb.Len() >= maxSize {
			out = append(out, strings.TrimSpace(sb.String()))
			sb.Reset()
		}
	}
	if sb.Len() > 0 {
		out = append(out, strings.TrimSpace(sb.String()))
	}
	return out
}
