package rank

import (
	"sort"
)

type Result struct {
	Name         string
	OnTimeTasks  int
	TimeoutTasks int
	Score        float32 // 表示 准时率，成功率，任务数
	Rank         int
}

// RankAndPaginateResults 排行
func RankAndPaginateResults(results []*Result, page, pageSize int, sortFlag bool) ([]*Result, error) {

	switch len(results) {
	case 0:
		return results, nil

	case 1:
		results[0].Rank = 1
		return results, nil
	}

	// 排序数据（按照 Score 降序排序） mysql查询出来的数据已经排序好了
	if sortFlag {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	}

	// 计算排名（使用 DENSE_RANK 算法）
	rank := 1
	prevRank := 1
	prevScore := results[0].Score
	results[0].Rank = rank

	for i := 0; i < len(results); i++ {
		if i > 0 {
			if results[i].Score < prevScore {
				rank += prevRank
				prevRank = 1
			} else if results[i].Score == prevScore {
				prevRank++
			} else {
				prevRank = 1
			}
		}
		results[i].Rank = rank
		prevScore = results[i].Score
	}
	// 判断是否要分页
	if page < 1 || pageSize < 1 {
		return results, nil
	}

	// 计算总页数
	totalPages := (len(results) + pageSize - 1) / pageSize

	// 检查页码是否有效
	if page > totalPages {
		return []*Result{}, nil
	}

	// 计算分页范围
	start := (page - 1) * pageSize
	end := start + pageSize

	if end > len(results) {
		end = len(results)
	}

	// 获取分页后的数据
	pagedResults := results[start:end]

	return pagedResults, nil
}
