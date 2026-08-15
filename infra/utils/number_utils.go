package utils

func ConvertFloat64ToFloat32(arr []float64) []float32 {
	// 数据类型转化
	ans := make([]float32, len(arr))
	for i, v := range arr {
		ans[i] = float32(v)
	}
	return ans
}
