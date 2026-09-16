package model


type AssetSource string

const (
	AssetSourceObject AssetSource = "object" // single presigned file (e.g. lambda binary)
	AssetSourceZip    AssetSource = "zip"    // presigned zip (folder/bucket export) — unpack after download
	AssetSourceInline AssetSource = "inline" // small payload embedded directly, base64
	AssetTypeGame     AssetType = "game"
	AssetTypeTemplate AssetType = "template"
	AssetTypeAgent    AssetType = "agent"
	AssetTypeScript   AssetType = "script"
)


type AssetType string



type CreatePresignedUploadURLRequest struct {
	UserID     string    `json:"user_id"`
	GameID     string    `json:"game_id,omitempty"`
	AssetID    string    `json:"asset_id"`
	AssetName  string    `json:"asset_name"`
	AssetType  AssetType `json:"asset_type"` // "game" | "template"
	Sha256     string    `json:"sha256"`
	Key        string    `json:"key"`
	BucketName string    `json:"bucket_name"` // this is the nameof the job/game/render job this asset belongs to like kalshi or ruto tracker

}
type CreatePresigneUploaddURLResponse struct {
	UploadURL string `json:"upload_url"`
}


type AddFileRequest struct{
	ProjectId string `json:"project_id"`
	Role string `json:"role"`
	OwnerID string `json:"owner_id"`
	FileName string `json:"file_name"`
	Sha256 string `json:"sha256"`
	Size int64 `json:"size"`
	ContentType  string `json:"content_type"`
}

type CreatePresignDownloadURLRequest struct {
	UserID        string `json:"user_id"`
	CorrelationID string `json:"correlation_id"`
	FileSha256    string `json:"sha256"`
	AssetID       string `json:"asset_id"`
	FileCount     int    `json:"file_count"`
}
type CreatePresignDownloadURLResponse struct {
	URL      string `json:"url"`
	BucketID string `json:"bucket_id"`
	Sha256   string `json:"sha256"`

	FileCount int `json:"file_count"` // zip or tar
}



type Asset struct {
	Name       string      `json:"name"`
	Source     AssetSource `json:"source"`
	URL        string      `json:"url,omitempty"`
	InlineData string      `json:"inline_data,omitempty"` // only for AssetSourceInline
	DestPath   string      `json:"dest_path"`             // where the agent places/unpacks it
	SHA256     string      `json:"sha256,omitempty"`
	Unpack     bool        `json:"unpack,omitempty"`     // true = unzip after download
	Executable bool        `json:"executable,omitempty"` // chmod +x after placing
}


type ArchiveByPrefixInputMessage struct {
	BucketID    string `json:"bucket_id"`
	Prefix      string `json:"prefix" binding:"required"`
	ArchiveName string `json:"archive_name" binding:"required"`
	Format      string `json:"format"` // zip or tar
	UserID      string `json:"user_id"`
}

type PresignedURLInfo struct {
	Name string `json:"name"`
	Role string `json:"role"`
	URL  string `json:"url"`
}