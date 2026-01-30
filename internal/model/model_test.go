package model

import "testing"

func TestNFTMetadata_IsImage(t *testing.T) {
	tests := []struct {
		name     string
		metadata *NFTMetadata
		want     bool
	}{
		{
			name:     "nil receiver returns false",
			metadata: nil,
			want:     false,
		},
		{
			name:     "empty struct returns false",
			metadata: &NFTMetadata{},
			want:     false,
		},
		{
			name: "ContentType image/png returns true",
			metadata: &NFTMetadata{
				ContentType: "image/png",
			},
			want: true,
		},
		{
			name: "ContentType image/jpeg returns true",
			metadata: &NFTMetadata{
				ContentType: "image/jpeg",
			},
			want: true,
		},
		{
			name: "ContentType image/gif returns true",
			metadata: &NFTMetadata{
				ContentType: "image/gif",
			},
			want: true,
		},
		{
			name: "ContentType application/pdf returns false",
			metadata: &NFTMetadata{
				ContentType: "application/pdf",
			},
			want: false,
		},
		{
			name: "ContentType text/markdown returns false",
			metadata: &NFTMetadata{
				ContentType: "text/markdown",
			},
			want: false,
		},
		{
			name: "ContentType shorter than 6 chars returns false",
			metadata: &NFTMetadata{
				ContentType: "img",
			},
			want: false,
		},
		{
			name: "ContentType exactly 'image' without slash returns false",
			metadata: &NFTMetadata{
				ContentType: "image",
			},
			want: false,
		},
		{
			name: "FileURL set returns false even with ImageURL",
			metadata: &NFTMetadata{
				ImageURL: "https://example.com/image.png",
				FileURL:  "https://example.com/doc.pdf",
			},
			want: false,
		},
		{
			name: "FullDescription exactly 500 chars with ImageURL returns true",
			metadata: &NFTMetadata{
				ImageURL:        "https://example.com/image.png",
				FullDescription: string(make([]byte, 500)),
			},
			want: true,
		},
		{
			name: "FullDescription 501 chars returns false even with ImageURL",
			metadata: &NFTMetadata{
				ImageURL:        "https://example.com/image.png",
				FullDescription: string(make([]byte, 501)),
			},
			want: false,
		},
		{
			name: "ImageURL set with no other fields returns true",
			metadata: &NFTMetadata{
				ImageURL: "https://example.com/image.png",
			},
			want: true,
		},
		{
			name: "only Name and Description set returns false",
			metadata: &NFTMetadata{
				Name:        "Test NFT",
				Description: "A test NFT",
			},
			want: false,
		},
		{
			name: "ContentType takes precedence over ImageURL",
			metadata: &NFTMetadata{
				ContentType: "text/plain",
				ImageURL:    "https://example.com/image.png",
			},
			want: false,
		},
		{
			name: "image ContentType returns true even without ImageURL",
			metadata: &NFTMetadata{
				ContentType: "image/webp",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.IsImage()
			if got != tt.want {
				t.Errorf("IsImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
