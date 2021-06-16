package util

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestRSAEncrypt(t *testing.T) {

	type Credibility struct {
		Score int
		Judge string
	}
	c:=Credibility{
		Score: 95,
		Judge: "excellent",
	}

	cB,_:=json.Marshal(c)
	pri,pub:=RSAGenerateKeys()
	pubB,_:=json.Marshal(pub)
	fmt.Println(string(pubB))

	data:=RSAEncrypt(cB,pub)
	fmt.Println("密文：",string(`Gjy��9\n0*�S�}�ɯ��}����#qok�Ҍ|_M1��OL�T��9���V��x����)���n0$\"�~�a{��OV���:{�\fY���\r\u000b_/�\f�y��m0j���Գ<�#fBя ���ʚ�]\"�]]�N�`))
	fmt.Println("加密结果：",string("[71,106,96,121,156,195,57,10,48,42,197,83,215,125,173,201,175,223,213,5,125,163,145,178,165,35,113,111,107,155,210,140,124,95,77,49,238,205,79,76,190,84,145,205,57,3,227,231,205,86,243,228,120,243,195,197,245,41,241,230,146,110,48,36,34,229,126,158,97,123,254,253,79,86,21,142,167,19,140,58,123,171,12,89,165,145,181,13,11,95,47,249,12,142,121,211,245,109,48,106,160,237,159,212,179,60,136,35,102,66,209,143,32,245,212,211,202,154,155,93,34,147,93,93,171,78,31,138]"))

	fmt.Println(`公钥：-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDWmTJ4S94/+XeOsJdq92NIP80L
BWgrl2FUvYqDLx8oemu+mdYMMtZFtVhcC0pa/TOSDR5zKo00rr6blof6sd1wnmGm
bFyqfTvmlDST0ZsBhk7UZvTRPeXIOixb+f1890SjOLOQAYv9WWgjzRgc3tf7ickG
odkN6eybztoprtft9QIDAQAB
-----END PUBLIC KEY-----`)

	d:=RSADecrypt(data,pri)
	fmt.Println("解密结果:",string(d))

}

func TestRSADecrypt(t *testing.T) {

}
