package meta

type ContractResponse struct {
	Read map[string]string
	Set  map[string]string
}
type ContractRequest struct {
	Method string
	Args   map[string]string
}

type ContractPost struct {
	Account string `json:"account"`
	PrivateKey string `json:"private_key"`
	PublicKey string `json:"public_key"`
	Code      string `json:"code"`
	Name      string `json:"name"`
}
