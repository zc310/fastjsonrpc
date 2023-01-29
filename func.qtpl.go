// Code generated by qtc from "func.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line func.qtpl:2
package fastjsonrpc

//line func.qtpl:2
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line func.qtpl:2
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line func.qtpl:2
func streamnewResult(qw422016 *qt422016.Writer, id, result []byte) {
//line func.qtpl:2
	qw422016.N().S(`{"jsonrpc":"2.0","result":`)
//line func.qtpl:5
	qw422016.N().Z(result)
//line func.qtpl:6
	if len(id) > 0 {
//line func.qtpl:6
		qw422016.N().S(`,"id":`)
//line func.qtpl:6
		qw422016.N().Z(id)
//line func.qtpl:6
	}
//line func.qtpl:6
	qw422016.N().S(`}`)
//line func.qtpl:8
}

//line func.qtpl:8
func writenewResult(qq422016 qtio422016.Writer, id, result []byte) {
//line func.qtpl:8
	qw422016 := qt422016.AcquireWriter(qq422016)
//line func.qtpl:8
	streamnewResult(qw422016, id, result)
//line func.qtpl:8
	qt422016.ReleaseWriter(qw422016)
//line func.qtpl:8
}

//line func.qtpl:8
func newResult(id, result []byte) string {
//line func.qtpl:8
	qb422016 := qt422016.AcquireByteBuffer()
//line func.qtpl:8
	writenewResult(qb422016, id, result)
//line func.qtpl:8
	qs422016 := string(qb422016.B)
//line func.qtpl:8
	qt422016.ReleaseByteBuffer(qb422016)
//line func.qtpl:8
	return qs422016
//line func.qtpl:8
}

//line func.qtpl:12
func streamnewError(qw422016 *qt422016.Writer, id []byte, code int, message string, data []byte) {
//line func.qtpl:12
	qw422016.N().S(`{"jsonrpc":"2.0","error":{"code":`)
//line func.qtpl:16
	qw422016.N().D(code)
//line func.qtpl:16
	qw422016.N().S(`,"message":"`)
//line func.qtpl:17
	qw422016.E().J(message)
//line func.qtpl:17
	qw422016.N().S(`"`)
//line func.qtpl:18
	if len(data) > 0 {
//line func.qtpl:18
		qw422016.N().S(`,"data":`)
//line func.qtpl:19
		qw422016.N().Z(data)
//line func.qtpl:20
	}
//line func.qtpl:20
	qw422016.N().S(`}`)
//line func.qtpl:22
	if len(id) > 0 {
//line func.qtpl:22
		qw422016.N().S(`,"id":`)
//line func.qtpl:22
		qw422016.N().Z(id)
//line func.qtpl:22
	}
//line func.qtpl:22
	qw422016.N().S(`}`)
//line func.qtpl:24
}

//line func.qtpl:24
func writenewError(qq422016 qtio422016.Writer, id []byte, code int, message string, data []byte) {
//line func.qtpl:24
	qw422016 := qt422016.AcquireWriter(qq422016)
//line func.qtpl:24
	streamnewError(qw422016, id, code, message, data)
//line func.qtpl:24
	qt422016.ReleaseWriter(qw422016)
//line func.qtpl:24
}

//line func.qtpl:24
func newError(id []byte, code int, message string, data []byte) string {
//line func.qtpl:24
	qb422016 := qt422016.AcquireByteBuffer()
//line func.qtpl:24
	writenewError(qb422016, id, code, message, data)
//line func.qtpl:24
	qs422016 := string(qb422016.B)
//line func.qtpl:24
	qt422016.ReleaseByteBuffer(qb422016)
//line func.qtpl:24
	return qs422016
//line func.qtpl:24
}
