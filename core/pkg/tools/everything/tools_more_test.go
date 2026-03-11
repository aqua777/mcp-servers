package everything

func (s *ToolsTestSuite) TestHandleSuccess_Internal() {
	res, err := handleSuccess("test string")
	s.Require().NoError(err)
	s.Require().NotNil(res)

	// Test the helper directly with non-string map
	res, err = handleSuccess(map[string]string{"foo": "bar"})
	s.Require().NoError(err)
	s.Require().NotNil(res)
}

func (s *ToolsTestSuite) TestHandleSuccess_JsonError() {
	// Generate an error by passing something that cannot be marshaled (e.g. channel)
	ch := make(chan int)
	res, err := handleSuccess(ch)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Require().True(res.IsError)
}
