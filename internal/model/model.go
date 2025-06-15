package model

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Link struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Tags []Tag  `json:"tags"`
}

// DTOs for incoming JSON

type IncomingTag struct {
	ID string `json:"id"`
}

type IncomingLink struct {
	URL  string        `json:"url"`
	Tags []IncomingTag `json:"tags"`
}
