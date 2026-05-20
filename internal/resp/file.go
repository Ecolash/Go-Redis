package resp

import "strconv"

func File(data []byte) []byte {
	header := "$" + strconv.Itoa(len(data)) + "\r\n"
	out := make([]byte, 0, len(header)+len(data))
	out = append(out, header...)
	out = append(out, data...)
	return out
}
