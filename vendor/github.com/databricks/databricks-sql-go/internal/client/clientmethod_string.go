// Code generated by "stringer -type=clientMethod"; DO NOT EDIT.

package client

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[unknown-0]
	_ = x[openSession-1]
	_ = x[closeSession-2]
	_ = x[fetchResults-3]
	_ = x[getResultSetMetadata-4]
	_ = x[executeStatement-5]
	_ = x[getOperationStatus-6]
	_ = x[closeOperation-7]
	_ = x[cancelOperation-8]
}

const _clientMethod_name = "unknownopenSessioncloseSessionfetchResultsgetResultSetMetadataexecuteStatementgetOperationStatuscloseOperationcancelOperation"

var _clientMethod_index = [...]uint8{0, 7, 18, 30, 42, 62, 78, 96, 110, 125}

func (i clientMethod) String() string {
	if i < 0 || i >= clientMethod(len(_clientMethod_index)-1) {
		return "clientMethod(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _clientMethod_name[_clientMethod_index[i]:_clientMethod_index[i+1]]
}
