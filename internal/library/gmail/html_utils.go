package gmail

import (
	"regexp"
	"strings"
)

// stripHTMLTags はHTMLタグを除去してプレーンテキストに変換します
func stripHTMLTags(html string) string {
	if html == "" {
		return ""
	}

	// HTMLエンティティをデコード
	html = decodeHTMLEntities(html)

	// HTMLタグを除去
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// 連続する空白文字を単一のスペースに変換
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// 前後の空白を除去
	text = strings.TrimSpace(text)

	return text
}

// decodeHTMLEntities は基本的なHTMLエンティティをデコードします
func decodeHTMLEntities(html string) string {
	// 基本的なHTMLエンティティのマップ
	entities := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&apos;": "'",
		"&nbsp;": " ",
		"&#39;":  "'",
		"&#34;":  "\"",
		"&#60;":  "<",
		"&#62;":  ">",
		"&#38;":  "&",
		"&#160;": " ",
	}

	result := html
	for entity, replacement := range entities {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	return result
}
