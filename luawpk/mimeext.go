package luawpk

const (
	jsoncontent = "application/json;charset=utf-8"
	htmlcontent = "text/html;charset=utf-8"
	csscontent  = "text/css;charset=utf-8"
	jscontent   = "text/javascript;charset=utf-8"
)

// MimeExt - map of files extensions and associated MIME types.
var MimeExt = map[string]string{
	// Web content
	".json":  jsoncontent,
	".html":  htmlcontent,
	".htm":   htmlcontent,
	".css":   csscontent,
	".js":    jscontent,
	".mjs":   jscontent,
	".map":   jsoncontent,
	".xml":   "text/xml",
	".xhtml": "application/xhtml+xml",
	// Text files
	".txt": "text/plain;charset=utf-8",
	".csv": "text/csv",
	".ics": "text/calendar",
	".sh":  "application/x-sh",
	".php": "application/x-httpd-php",
	".csh": "application/x-csh",
	// Office files
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".odp":  "application/vnd.oasis.opendocument.presentation",
	".rtf":  "application/rtf",
	".abw":  "application/x-abiword",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".vsd":  "application/vnd.visio",
	// Application files
	".bin":  "application/octet-stream",
	".pdf":  "application/pdf",
	".djvu": "image/x-djvu",
	".djv":  "image/x-djvu",
	".swf":  "application/x-shockwave-flash",
	".pem":  "application/x-pem-file",
	".cer":  "application/pkix-cert",
	".crt":  "application/x-x509-ca-cert",
	".m3u":  "application/mpegurl",
	".m3u8": "application/mpegurl",
	".wpl":  "application/vnd.ms-wpl",
	".pls":  "audio/x-scpls",
	".asx":  "video/x-ms-asf",
	".xspf": "application/xspf+xml",
	// Image types
	".tga":  "image/targa",
	".bmp":  "image/bmp",
	".dib":  "image/bmp",
	".rle":  "image/bmp",
	".gif":  "image/gif",
	".png":  "image/png",
	".apng": "image/apng",
	".jpg":  "image/jpeg",
	".jpe":  "image/jpeg",
	".jpeg": "image/jpeg",
	".jfif": "image/jpeg",
	".jp2":  "image/jp2",
	".jpg2": "image/jp2",
	".jpx":  "image/jp2",
	".jpm":  "image/jpm",
	".jxr":  "image/jxr",
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
	".psb":  "image/photoshop",
	// Audio types
	".aac":  "audio/aac",
	".m4a":  "audio/x-m4a",
	".aif":  "audio/aiff",
	".mpa":  "audio/mpeg",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".wma":  "audio/x-ms-wma",
	".weba": "audio/webm",
	".oga":  "audio/ogg",
	".ogg":  "audio/ogg",
	".opus": "audio/opus",
	".flac": "audio/x-flac",
	".mka":  "audio/x-matroska",
	".ra":   "audio/vnd.rn-realaudio",
	".mid":  "audio/mid",
	".midi": "audio/mid",
	".cda":  "application/x-cdf",
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
	".3g2":  "video/3gpp2",
	".mkv":  "video/x-matroska",
	".mov":  "video/quicktime",
	".ogv":  "video/ogg",
	".ogx":  "application/ogg",
	// Archive types
	".zip": "application/zip",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
	".bz":  "application/x-bzip",
	".bz2": "application/x-bzip2",
	".rar": "application/vnd.rar",
	".jar": "application/java-archive",
	".7z":  "application/x-7z-compressed",
	".arc": "application/x-freearc",
	// Fonts types
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".eot":   "application/vnd.ms-fontobject",
}

// The End.
