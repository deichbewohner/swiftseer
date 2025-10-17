package models

type UploadResponse struct {
	UUID  string       `json:"uuid"`
	Files []UploadFile `json:"files"`
}

type UploadFile struct {
	UUID     string `json:"uuid"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

func (u *UploadResponse) GetFileUUID() string {
	if len(u.Files) > 0 {
		return u.Files[0].UUID
	}
	return ""
}
