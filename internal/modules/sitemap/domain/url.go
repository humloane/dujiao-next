package domain

// URL 是 Sitemap 中的单个可索引地址。
type URL struct {
	Location        string
	LastModified    string
	ChangeFrequency string
	Priority        string
	// Alternates 是 hreflang -> URL 映射，输出为 sitemap 的 xhtml:link。
	Alternates map[string]string
}
