package util

import (
	"JiraConnector/connector/pb"
	"fmt"
)

func ValidateParams(request *pb.ProjectsRequest) (*pb.ProjectsRequest, error) {
	limit, err := Validate(request.Limit, func(int32) bool {
		if request.Limit < 0 {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	page, err := Validate(request.Page, func(int32) bool {
		if request.Page < 0 {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return &pb.ProjectsRequest{
		Limit:  limit,
		Page:   page,
		Search: request.Search,
	}, nil
}

func Validate[T any](input T, f func(T) bool) (T, error) {
	if !f(input) {
		var zero T
		return zero, fmt.Errorf("invalid value in response: %v", input)
	}
	return input, nil
}

func Filter[T any](a []T, f func(T) bool) (b []T) {
	for _, c := range a {
		if f(c) {
			b = append(b, c)
		}
	}
	return
}
