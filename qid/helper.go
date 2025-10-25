package qid

import (
	"context"
	"fmt"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/utils"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func (q *Qid) readPipeline(ctx context.Context, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
	ectx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"string": cty.StringVal("string"),
			"number": cty.StringVal("number"),
			"bool":   cty.StringVal("bool"),
		},
	}
	var pvars pipeline.Variables
	err := hclsimple.Decode("pipeline.hcl", rpp, ectx, &pvars)
	if err != nil {
		return nil, fmt.Errorf("failed to Decode Pipeline config: %w", err)
	}

	ecvars := make(map[string]cty.Value)
	for _, v := range pvars.Variables {
		switch v.Type {
		case "string":
			if mv, ok := vars[v.Name]; ok {
				s, ok := mv.(string)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'string'", v.Name)
				}
				ecvars[v.Name] = cty.StringVal(s)
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'string'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var s string
				err = gocty.FromCtyValue(ctyv, &s)
				if err != nil {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'string'", v.Name)
				}
				ecvars[v.Name] = cty.StringVal(s)
			}
		case "number":
			if mv, ok := vars[v.Name]; ok {
				n, ok := mv.(float64)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'number'", v.Name)
				}
				ecvars[v.Name] = cty.NumberVal(big.NewFloat(n))
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'number'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var n float64
				err = gocty.FromCtyValue(ctyv, &n)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'number'", v.Name)
				}
				ecvars[v.Name] = cty.NumberVal(big.NewFloat(n))
			}
		case "bool":
			if mv, ok := vars[v.Name]; ok {
				b, ok := mv.(bool)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'bool'", v.Name)
				}
				ecvars[v.Name] = cty.BoolVal(b)
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'bool'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var b bool
				err = gocty.FromCtyValue(ctyv, &b)
				if err != nil {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'bool'", v.Name)
				}
				ecvars[v.Name] = cty.BoolVal(b)
			}
		}
	}
	ectx = &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(ecvars),
		},
	}

	var pp pipeline.Pipeline
	err = hclsimple.Decode("pipeline.hcl", rpp, ectx, &pp)
	if err != nil {
		for _, e := range err.(hcl.Diagnostics).Errs() {
			spew.Dump(e)
		}
		return nil, fmt.Errorf("failed to Decode Pipeline config: %w", err)
	}
	for i, r := range pp.Resources {
		pp.Resources[i].Canonical = utils.ResourceCanonical(r.Type, r.Name)
	}
	return &pp, nil
}
