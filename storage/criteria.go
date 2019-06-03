package storage

import "github.com/Peripli/service-manager/pkg/query"

type CriteriaContainer struct {
	QueryCriteria []query.Criterion
	ListCriteria  []ListCriteria
}

func ByCriteria(criteria ...interface{}) CriteriaContainer {
	result := CriteriaContainer{}
	for _, c := range criteria {
		switch crit := c.(type) {
		case query.Criterion:
			result.QueryCriteria = append(result.QueryCriteria, crit)
		case []query.Criterion:
			result.QueryCriteria = append(result.QueryCriteria, crit...)
		case ListCriteria:
			result.ListCriteria = append(result.ListCriteria, crit)
		case []ListCriteria:
			result.ListCriteria = append(result.ListCriteria, crit...)
		}
	}
	return result
}

func MergeCriteriaContainers(containers ...CriteriaContainer) CriteriaContainer {
	result := CriteriaContainer{}
	for _, c := range containers {
		result.QueryCriteria = append(result.QueryCriteria, c.QueryCriteria...)
		result.ListCriteria = append(result.ListCriteria, c.ListCriteria...)
	}
	return result
}
