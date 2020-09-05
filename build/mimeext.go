package main

const (
	jsoncontent = "application/json;charset=utf-8"
	htmlcontent = "text/html;charset=utf-8"
	csscontent  = "text/css;charset=utf-8"
	jscontent   = "text/javascript;charset=utf-8"
)

var mimeext = map[string]string{
	// Common text content
	".json": jsoncontent,
	".html": htmlcontent,
	".htm":  htmlcontent,
	".css":  csscontent,
	".js":   jscontent,
	".mjs":  jscontent,
	".map":  jsoncontent,
	".txt":  "text/plain;charset=utf-8",
	".pdf":  "application/pdf",
	".swf":  "application/x-shockwave-flash",
	".doc":  "application/msword",
	".xls":  "application/vnd.ms-excel",
	".csv":  "application/vnd.ms-excel",
	// Image types
	".tga":  "image/targa",
	".bmp":  "image/bmp",
	".dib":  "image/bmp",
	".gif":  "image/gif",
	".png":  "image/png",
	".apng": "image/apng",
	".jpg":  "image/jpeg",
	".jpe":  "image/jpeg",
	".jpeg": "image/jpeg",
	".jfif": "image/jpeg",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".cur":  "image/x-icon",
	".dds":  "image/vnd.ms-dds",
	".dng":  "image/DNG",
	".pxn":  "image/PXN",
	".wmf":  "image/x-wmf",
	".psd":  "image/photoshop",
	// Audio types
	".aac":  "audio/aac",
	".aif":  "audio/aiff",
	".mpa":  "audio/mpeg",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".wma":  "audio/x-ms-wma",
	".weba": "audio/webm",
	".oga":  "audio/ogg",
	".ogg":  "audio/ogg",
	".opus": "audio/ogg",
	".flac": "audio/x-flac",
	".mka":  "audio/x-matroska",
	".ra":   "audio/vnd.rn-realaudio",
	".mid":  "audio/mid",
	".midi": "audio/mid",
	// Video types, multimedia containers
	".avi":  "video/avi",
	".mpe":  "video/mpeg",
	".mpg":  "video/mpeg",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".wmv":  "video/x-ms-wmv",
	".wmx":  "video/x-ms-wmx",
	".flv":  "video/x-flv",
	".3gp":  "video/3gpp",
	".mkv":  "video/x-matroska",
	".mov":  "video/quicktime",
	".ogx":  "video/ogg",
	// Fonts types
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

// The End.
