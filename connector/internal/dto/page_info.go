package dto

type PageInfo struct {
	PageCount     int32 `json:"pageCount"`
	CurrentPage   int32 `json:"currentPage"`
	ProjectsCount int32 `json:"projectsCount"`
}
