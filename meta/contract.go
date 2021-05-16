package meta


type ContractResponse struct {
	Read map[string]string
	Set  map[string]string
}
type ContractRequest struct {
	Method string
	Args   map[string]string
}
