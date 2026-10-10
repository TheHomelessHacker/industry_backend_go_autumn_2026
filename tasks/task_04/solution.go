package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) <= 1 {
		return Stats{}
	}
	diff := nums[1] - nums[0]
	count, sum, min, max := 1, diff, diff, diff

	for index := 1; index < len(nums) - 1; index++ {
		diff = nums[index + 1] - nums[index]
		count += 1
		sum += diff
		if diff > max { max = diff }
		if diff < min { min = diff }
	}
	return Stats{Count: count, Sum: sum, Min: min, Max: max}
}
