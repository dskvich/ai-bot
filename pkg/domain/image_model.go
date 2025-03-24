package domain

const (
	DallE2Model = "dall-e-2"
	DallE3Model = "dall-e-3"
)

type ImageSize string

const (
	Size256x256   ImageSize = "256x256"
	Size512x512   ImageSize = "512x512"
	Size1024x1024 ImageSize = "1024x1024"
	Size1024x1792 ImageSize = "1024x1792"
	Size1792x1024 ImageSize = "1792x1024"
)

type ImageQuality string

const (
	QualityStandard ImageQuality = "standard"
	QualityHD       ImageQuality = "hd"
)
