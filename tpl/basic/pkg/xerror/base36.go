/**
 * @Author:
 * @Date: 2024-03-28 15:08
 * @Desc: 错误 base36 编码
 */

package xerror

var (
	base36 = []byte{
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J',
		'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T',
		'U', 'V', 'W', 'X', 'Y', 'Z',
	}

	uint8Index [256]uint64
	pow36Index = []uint64{
		1,
		36,
		1296,
		46656,
		1679616,
		60466176,
		2176782336,
		78364164096,
		2821109907456,
		101559956668416,
		3656158440062976,
		131621703842267136,
		4738381338321616896,
	}
)

func init() {
	for i := 0; i < 256; i++ {
		switch {
		case i >= '0' && i <= '9':
			uint8Index[i] = uint64(i - '0')
		case i >= 'A' && i <= 'Z':
			uint8Index[i] = uint64(i - 'A' + 10)
		case i >= 'a' && i <= 'z':
			uint8Index[i] = uint64(i - 'a' + 10)
		default:
			uint8Index[i] = 0
		}
	}
}

// encodes a number to base36.
func base36Encode(value uint64) string {
	var res [16]byte
	var i int
	for i = len(res) - 1; ; i-- {
		res[i] = base36[value%36]
		value /= 36
		if value == 0 {
			break
		}
	}
	return string(res[i:])
}

// decodes a base36-encoded string.
func base36Decode(s string) uint64 {
	if len(s) > 13 {
		s = s[:13]
	}
	res := uint64(0)
	l := len(s) - 1
	for idx := 0; idx < len(s); idx++ {
		c := s[l-idx]
		res += uint8Index[c] * pow36Index[idx]
	}
	return res
}
