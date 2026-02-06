package tracker

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// Bencode 编码器 - 实现 BitTorrent BEP-0003 标准
// 规范: https://www.bittorrent.org/beps/bep_0003.html

// EncodeString 编码字符串: <长度>:<内容>
// 例如: "spam" -> "4:spam"
func EncodeString(s string) []byte {
	return []byte(fmt.Sprintf("%d:%s", len(s), s))
}

// EncodeBytes 编码字节数组
func EncodeBytes(b []byte) []byte {
	return []byte(fmt.Sprintf("%d:%s", len(b), string(b)))
}

// EncodeInt 编码整数: i<数字>e
// 例如: 42 -> "i42e"
func EncodeInt(n int64) []byte {
	return []byte(fmt.Sprintf("i%de", n))
}

// EncodeList 编码列表: l<元素>e
// 例如: ["spam", "eggs"] -> "l4:spam4:eggse"
func EncodeList(items [][]byte) []byte {
	buf := bytes.NewBuffer([]byte("l"))
	for _, item := range items {
		buf.Write(item)
	}
	buf.WriteByte('e')
	return buf.Bytes()
}

// EncodeDict 编码字典: d<key><value>e
// 例如: {"key": "value"} -> "d3:key5:valuee"
// 注意: 键必须按字典序排序
func EncodeDict(dict map[string][]byte) []byte {
	// 获取所有键并排序
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := bytes.NewBuffer([]byte("d"))
	for _, key := range keys {
		buf.Write(EncodeString(key))
		buf.Write(dict[key])
	}
	buf.WriteByte('e')
	return buf.Bytes()
}

// DecodeString 解码字符串（简单实现）
func DecodeString(data []byte) (string, int, error) {
	colonIndex := bytes.IndexByte(data, ':')
	if colonIndex == -1 {
		return "", 0, fmt.Errorf("invalid string format")
	}

	length, err := strconv.Atoi(string(data[:colonIndex]))
	if err != nil {
		return "", 0, fmt.Errorf("invalid string length: %w", err)
	}

	start := colonIndex + 1
	end := start + length
	if end > len(data) {
		return "", 0, fmt.Errorf("string length exceeds data")
	}

	return string(data[start:end]), end, nil
}
