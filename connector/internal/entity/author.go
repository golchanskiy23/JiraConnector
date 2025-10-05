package entity

type ChangeLog struct {
	Count     int       `json:"total"`
	Histories []History `json:"histories"`
}

type History struct {
	Author      Author         `json:"author"`
	CreatedTime string         `json:"created"`
	Items       []StatusChange `json:"items"`
}

type StatusChange struct {
	FromStatus string `json:"fromString"`
	ToStatus   string `json:"toString"`
}

type Author struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}
