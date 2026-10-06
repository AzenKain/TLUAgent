//go:build cgo

package embedding

import (
	ort "github.com/yalue/onnxruntime_go"
)

// ortValue wraps an ONNX Runtime tensor value behind a cgo-free interface.
type ortValue interface {
	Destroy() error
	floatData() ([]float32, []int64, bool)
}

// ortSession wraps a dynamic ONNX Runtime session.
type ortSession interface {
	Run(inputs, outputs []ortValue) error
	Destroy() error
}

type ortIOInfo struct {
	Name string
}

type cgoValue struct {
	v ort.Value
}

func (c cgoValue) Destroy() error {
	return c.v.Destroy()
}

func (c cgoValue) floatData() ([]float32, []int64, bool) {
	tensor, ok := c.v.(*ort.Tensor[float32])
	if !ok {
		return nil, nil, false
	}
	return tensor.GetData(), tensor.GetShape(), true
}

type cgoSession struct {
	s *ort.DynamicAdvancedSession
}

func (c *cgoSession) Run(inputs, outputs []ortValue) error {
	valuesIn := make([]ort.Value, len(inputs))
	for i, v := range inputs {
		valuesIn[i] = v.(cgoValue).v
	}
	valuesOut := make([]ort.Value, len(outputs))
	for i := range outputs {
		valuesOut[i] = nil
	}
	if err := c.s.Run(valuesIn, valuesOut); err != nil {
		return err
	}
	for i := range outputs {
		if valuesOut[i] != nil {
			outputs[i] = cgoValue{valuesOut[i]}
		}
	}
	return nil
}

func (c *cgoSession) Destroy() error {
	return c.s.Destroy()
}

func ortSetLibraryPath(path string) {
	ort.SetSharedLibraryPath(path)
}

func ortInitializeEnvironment() error {
	return ort.InitializeEnvironment()
}

func ortNewDynamicSession(modelPath string, inputNames, outputNames []string) (ortSession, error) {
	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, err
	}
	return &cgoSession{s: session}, nil
}

func ortNewInt64Tensor(dims []int64, data []int64) (ortValue, error) {
	tensor, err := ort.NewTensor(ort.NewShape(dims...), data)
	if err != nil {
		return nil, err
	}
	return cgoValue{tensor}, nil
}

func ortGetIOInfo(modelPath string) ([]ortIOInfo, []ortIOInfo, error) {
	inputs, outputs, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		return nil, nil, err
	}
	inInfos := make([]ortIOInfo, len(inputs))
	for i, info := range inputs {
		inInfos[i] = ortIOInfo{Name: info.Name}
	}
	outInfos := make([]ortIOInfo, len(outputs))
	for i, info := range outputs {
		outInfos[i] = ortIOInfo{Name: info.Name}
	}
	return inInfos, outInfos, nil
}

func requireCgoRuntime() error {
	return nil
}
