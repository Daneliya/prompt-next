package desktop

import (
	"errors"
	"slices"

	"github.com/Daneliya/prompt-next/internal/library"
)

func (s *PromptService) StageImageUpload(id, encoded string) error {
	if len(id) == 0 || len(id) > 64 {
		return errors.New("图片上传编号无效")
	}
	data, err := library.DecodeImageUpload(encoded)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("图片文件为空")
	}
	s.imageUploadMu.Lock()
	defer s.imageUploadMu.Unlock()
	previous := len(s.imageUploads[id])
	total := s.imageUploadBytes - previous + len(data)
	if total > library.MaxImageUploadBytes {
		return errors.New(localizedLabel(ReadLanguage(), "每次保存的新图片合计须在 80 MB 以内", "New images must total 80 MB or less per save"))
	}
	if s.imageUploads == nil {
		s.imageUploads = make(map[string][]byte)
	}
	s.imageUploads[id] = data
	s.imageUploadBytes = total
	return nil
}

func (s *PromptService) DiscardImageUploads(ids []string) {
	s.imageUploadMu.Lock()
	defer s.imageUploadMu.Unlock()
	for _, id := range ids {
		s.removeImageUpload(id)
	}
}

func (s *PromptService) removeImageUpload(id string) {
	s.imageUploadBytes -= len(s.imageUploads[id])
	delete(s.imageUploads, id)
}

func (s *PromptService) resolveImageUploads(attachments []library.ImageInput) ([]library.ImageInput, error) {
	s.imageUploadMu.Lock()
	defer s.imageUploadMu.Unlock()
	resolved := slices.Clone(attachments)
	for i := range resolved {
		input := &resolved[i]
		if input.UploadID == "" {
			continue
		}
		input.UploadBytes = s.imageUploads[input.UploadID]
		if input.UploadBytes == nil || input.Data != "" || input.Name != "" {
			return nil, errors.New(localizedLabel(ReadLanguage(), "图片上传失效，请重试保存", "The image upload expired. Save again to retry."))
		}
	}
	for _, input := range resolved {
		if input.UploadID != "" {
			s.removeImageUpload(input.UploadID)
		}
	}
	return resolved, nil
}
