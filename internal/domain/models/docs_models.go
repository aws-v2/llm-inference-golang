package model



type DocItem struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type DocCategory struct {
	Title string    `json:"title"`
	Items []DocItem `json:"items"`
}

type DocManifest struct {
	Service    string        `json:"service"`
	APIVersion string        `json:"apiVersion,omitempty"`
	Scope      string        `json:"scope"`
	Internal   []DocCategory `json:"internal,omitempty"`
	Public     []DocCategory `json:"public,omitempty"`
}

type Metadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	LastUpdated string   `json:"lastUpdated"`
	Tags        []string `json:"tags"`
}

type DocResponse struct {
	Metadata Metadata `json:"metadata"`
	Content  string   `json:"content"`
}
