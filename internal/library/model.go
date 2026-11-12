package library

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

type Prompt struct {
	Number          int64    `json:"number"`
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	Title           string   `json:"title"`
	English         string   `json:"english"`
	Images          []string `json:"images"`
	ReferenceImages []string `json:"referenceImages,omitempty"`
	Body            string   `json:"body"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	Favorite        bool     `json:"favorite"`
	Author          string   `json:"author"`
	AuthorURL       string   `json:"authorUrl"`
	VideoURL        string   `json:"videoUrl"`
	Video           string   `json:"video,omitempty"`
	Videos          []string `json:"videos"`
	Negative        string   `json:"negative"`
	Model           string   `json:"model"`
	Models          []string `json:"models"`
	Parameters      string   `json:"parameters"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

func (p *Prompt) validate() error {
	p.Title = strings.TrimSpace(p.Title)
	p.Category = strings.TrimSpace(p.Category)
	if p.Kind != "persona" && p.Kind != "art" && p.Kind != "text" && p.Kind != "video" {
		return errors.New("提示词库类型无效")
	}
	if p.Title == "" || (strings.TrimSpace(p.Body) == "" && (!supportsMedia(p.Kind) || strings.TrimSpace(p.English) == "")) {
		return errors.New("请填写标题和提示词正文")
	}
	p.Author = strings.TrimSpace(p.Author)
	p.AuthorURL = strings.TrimSpace(p.AuthorURL)
	p.VideoURL = strings.TrimSpace(p.VideoURL)
	if p.AuthorURL != "" && !ValidWebLink(p.AuthorURL) {
		return errors.New("提示词原链接须为有效的 HTTP 或 HTTPS 网址")
	}
	if p.VideoURL != "" && (p.Kind != "video" || !ValidWebLink(p.VideoURL)) {
		return errors.New("参考视频链接须为有效的 HTTP 或 HTTPS 网址")
	}
	p.Videos = cleanTags(p.videoNames())
	p.Video = ""
	if len(p.Videos) > MaxPromptVideos || (p.Kind != "video" && len(p.Videos) > 0) {
		return errors.New("每条视频提示词最多可附加 12 段视频")
	}
	for _, name := range p.Videos {
		if !videoName.MatchString(name) {
			return errors.New("视频引用无效")
		}
	}
	p.Tags = cleanTags(p.Tags)
	p.normalizeModels()
	p.Images = cleanTags(p.Images)
	p.ReferenceImages = cleanTags(p.ReferenceImages)
	for _, name := range p.imageNames() {
		if !supportsMedia(p.Kind) || !imageName.MatchString(name) {
			return errors.New("图片引用无效")
		}
	}
	return nil
}

func (p Prompt) imageNames() []string {
	return slices.Concat(p.Images, p.ReferenceImages)
}

func (p Prompt) videoNames() []string {
	if p.Videos == nil && p.Video != "" {
		return []string{p.Video}
	}
	return p.Videos
}

func supportsMedia(kind string) bool {
	return kind == "art" || kind == "video"
}

func ValidWebLink(link string) bool {
	parsed, err := url.Parse(link)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Hostname() != "" && parsed.User == nil
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
