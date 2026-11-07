package library

import (
	"errors"
	"strings"
)

type Prompt struct {
	Number      int64    `json:"number"`
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	English     string   `json:"english"`
	Images      []string `json:"images"`
	Body        string   `json:"body"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Favorite    bool     `json:"favorite"`
	Author      string   `json:"author"`
	Negative    string   `json:"negative"`
	Model       string   `json:"model"`
	Models      []string `json:"models"`
	Parameters  string   `json:"parameters"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

func (p *Prompt) validate() error {
	p.Title = strings.TrimSpace(p.Title)
	p.Category = strings.TrimSpace(p.Category)
	if p.Kind != "persona" && p.Kind != "art" {
		return errors.New("提示词库类型无效")
	}
	if p.Title == "" || (strings.TrimSpace(p.Body) == "" && (p.Kind != "art" || strings.TrimSpace(p.English) == "")) {
		return errors.New("请填写标题和提示词正文")
	}
	p.Tags = cleanTags(p.Tags)
	p.normalizeModels()
	p.Images = cleanTags(p.Images)
	for _, name := range p.Images {
		if p.Kind != "art" || !imageName.MatchString(name) {
			return errors.New("图片引用无效")
		}
	}
	return nil
}

func (p *Prompt) normalizeModels() {
	if p.Models == nil && p.Model != "" {
		p.Models = []string{p.Model}
	}
	p.Models = cleanTags(p.Models)
	p.Model = ""
	if len(p.Models) > 0 {
		p.Model = p.Models[0]
	}
}

func cleanTags(values []string) []string {
	tags := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		tag := strings.TrimSpace(value)
		if tag != "" && !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return tags
}
