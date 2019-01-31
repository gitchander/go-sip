package transport

func cloneBytes(a []byte) []byte {
	b := make([]byte, len(a))
	copy(b, a)
	return b
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
