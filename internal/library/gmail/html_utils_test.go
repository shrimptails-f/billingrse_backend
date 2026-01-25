package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空文字列の場合に空文字列を返すこと",
			input:    "",
			expected: "",
		},
		{
			name:     "プレーンテキストの場合にそのまま返すこと",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "基本的なHTMLタグを除去すること",
			input:    "<div>Hello, World!</div>",
			expected: "Hello, World!",
		},
		{
			name:     "複数のHTMLタグを除去すること",
			input:    "<div><p>Hello, <strong>World</strong>!</p></div>",
			expected: "Hello, World!",
		},
		{
			name:     "スタイル属性付きのHTMLタグを除去すること",
			input:    `<div style="color: red;"><font style="font-size: 12px;">Hello, World!</font></div>`,
			expected: "Hello, World!",
		},
		{
			name:     "HTMLエンティティをデコードすること",
			input:    "&lt;div&gt;Hello &amp; World!&lt;/div&gt;",
			expected: "Hello & World!",
		},
		{
			name:     "連続する空白文字を単一のスペースに変換すること",
			input:    "<div>Hello,\n\t   World!</div>",
			expected: "Hello, World!",
		},
		{
			name:     "改行とタブを含むHTMLを正しく処理すること",
			input:    "<div>\n\t<p>Hello,</p>\n\t<p>World!</p>\n</div>",
			expected: "Hello, World!",
		},
		{
			name:     "ネストしたHTMLタグを除去すること",
			input:    "<div><span><em><strong>Hello, World!</strong></em></span></div>",
			expected: "Hello, World!",
		},
		{
			name:     "自己終了タグを除去すること",
			input:    "Hello,<br/>World!<hr/>",
			expected: "Hello,World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空文字列の場合に空文字列を返すこと",
			input:    "",
			expected: "",
		},
		{
			name:     "HTMLエンティティがない場合にそのまま返すこと",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "基本的なHTMLエンティティをデコードすること",
			input:    "&amp;&lt;&gt;&quot;&apos;",
			expected: "&<>\"'",
		},
		{
			name:     "数値文字参照をデコードすること",
			input:    "&#39;&#34;&#60;&#62;&#38;",
			expected: "'\"<>&",
		},
		{
			name:     "ノーブレークスペースをデコードすること",
			input:    "Hello&nbsp;World&#160;!",
			expected: "Hello World !",
		},
		{
			name:     "複数のHTMLエンティティを含む文字列をデコードすること",
			input:    "Hello &amp; World &lt;test&gt; &quot;example&quot;",
			expected: "Hello & World <test> \"example\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeHTMLEntities(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
