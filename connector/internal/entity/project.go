package entity

type Project struct {
	ID          string
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"self"`
	Description string `json:"description"`
}
