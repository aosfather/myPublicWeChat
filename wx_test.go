package main

import "testing"

func TestAES_Init(t *testing.T) {
	encryted := &ApplicationEncrypt{}
	encryted.Init("aosfather", "wxafa7416fdcb42e36", "WB7S1I4ZDh2zrYw1eUbnL2jxs8glTezjp1HPkUIcYNq")
	t.Log(encryted.VerifyURL("d12a34ea341c91f0c9a15d8c550e01393942724c", "1536832575", "10705336", "208569820262595839"))
}
