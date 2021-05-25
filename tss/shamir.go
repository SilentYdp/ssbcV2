package tss

import "go.dedis.ch/kyber/v3"

type (
	ConstantTerms struct {
		Cons []kyber.Scalar
	}
	SharedSecret struct {
		X      int64
		Secret kyber.Scalar
	}
)

// NewRandomConstantTerms generates 多項式の定数項のリスト
func NewRandomConstantTerms(cnt int) *ConstantTerms {
	cons := make([]kyber.Scalar, cnt)
	for i := range cons {
		cons[i] = curve.Scalar().Pick(curve.RandomStream())
	}

	return &ConstantTerms{
		Cons: cons,
	}
}

func (t *ConstantTerms) Len() int {
	return len(t.Cons)
}

// GetSecret returns 秘匿情報
// e.g. f(x) = a + bx + cx^2 の場合 a が秘匿情報となる
//      -> 多項式の x に 0 を適用した値
func (t *ConstantTerms) GetSecret() kyber.Scalar {
	if t.Len() == 0 {
		return nil
	}
	return t.Cons[0]
}

// CalcShare calculates shared secret
// 個々のユーザーにシェアする秘匿情報
// -> 多項式の x に 0 以外の値を適用した値
func (t *ConstantTerms) CalcShare(x int64) *SharedSecret {
	if x == 0 {
		panic(`apply 0 then derive secret value`)
	}
	secret := curve.Scalar().Zero()

	xScalar := curve.Scalar().SetInt64(x)
	for exp, c := range t.Cons {
		s := c
		for i := 1; i <= exp; i++ {
			s = curve.Scalar().Mul(s, xScalar)
		}
		secret = curve.Scalar().Add(secret, s)
	}
	return &SharedSecret{
		X:      x,
		Secret: secret,
	}
}

// Solve calculates interpolated value
// 集めたシェアからラグランジュの補間公式を利用して元の情報を復元する
func Solve(sharedList ...*SharedSecret) kyber.Scalar {
	res := curve.Scalar().Zero()
	for _, iS := range sharedList {
		numer := curve.Scalar().SetInt64(1)
		denom := curve.Scalar().SetInt64(1)
		for _, jS := range sharedList {
			if iS.X == jS.X {
				continue
			}
			numer = curve.Scalar().Mul(
				numer,
				curve.Scalar().Mul(
					curve.Scalar().SetInt64(-1),
					curve.Scalar().SetInt64(jS.X),
				))
			denom = curve.Scalar().Mul(
				denom,
				curve.Scalar().Sub(
					curve.Scalar().SetInt64(iS.X),
					curve.Scalar().SetInt64(jS.X),
				))
		}
		res = curve.Scalar().Add(
			res,
			curve.Scalar().Mul(
				curve.Scalar().Div(numer, denom),
				iS.Secret,
			),
		)
	}

	return res
}
