package dynaml

import (
	"container/list"
	"strconv"
	"strings"
	
	"github.com/cloudfoundry-incubator/spiff/yaml"
	"github.com/cloudfoundry-incubator/spiff/debug"
)

/////////////////////////////////////////////////////////////////
// internal helper nodes
// used during expression parsing, they will never be contained in finally
// parsed expression trees
/////////////////////////////////////////////////////////////////

type helperNode struct { }

func (e helperNode) Evaluate(binding Binding) (yaml.Node, EvaluationInfo, bool) {
	panic("not intended to be evaluated")
}

/*
 * internal helper expression node to gather expression lists
 * used for list constants and call argument lists during expression parsing
 */
type expressionListHelper struct {
	helperNode
	list []Expression
}

/*
 * internal helper expression node to gather expression lists
 * used for list constants and call argument lists during expression parsing
 */
type nameHelper struct {
	helperNode
	name string
}

/////////////////////////////////////////////////////////////////
// Parsing
/////////////////////////////////////////////////////////////////

func Parse(source string, path []string) (Expression, error) {
	grammar := &DynamlGrammar{Buffer: source}
	grammar.Init()

	err := grammar.Parse()
	if err != nil {
		return nil, err
	}

	return buildExpression(grammar, path), nil
}

func buildExpression(grammar *DynamlGrammar, path []string) Expression {
	tokens := &tokenStack{}
    replace:= false
	
	for token := range grammar.Tokens() {
		contents := grammar.Buffer[token.begin:token.end]

		switch token.pegRule {
		case ruleDynaml:
			return tokens.Pop()
		case ruleAuto:
			tokens.Push(AutoExpr{path})
		case ruleMerge:
			replace = false
		case ruleSimpleMerge:
			debug.Debug("*** rule simple merge\n")
			tokens.Push(MergeExpr{path,false,replace})
		case ruleRefMerge:
			debug.Debug("*** rule ref merge\n")
			rhs := tokens.Pop()
			tokens.Push(MergeExpr{rhs.(ReferenceExpr).Path,true,replace})
		case ruleReplace:
			replace = true
		case ruleReference:
			tokens.Push(ReferenceExpr{strings.Split(contents, ".")})
		case ruleInteger:
			val, err := strconv.ParseInt(contents, 10, 64)
			if err != nil {
				panic(err)
			}

			tokens.Push(IntegerExpr{val})
		case ruleNil:
			tokens.Push(NilExpr{})
		case ruleBoolean:
			tokens.Push(BooleanExpr{contents == "true"})
		case ruleString:
			val := strings.Replace(contents[1:len(contents)-1], `\"`, `"`, -1)
			tokens.Push(StringExpr{val})
		case ruleOr:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(OrExpr{A: lhs, B: rhs})
		case ruleConcatenation:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(ConcatenationExpr{A: lhs, B: rhs})
		case ruleAddition:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(AdditionExpr{A: lhs, B: rhs})
		case ruleSubtraction:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(SubtractionExpr{A: lhs, B: rhs})
		case ruleMultiplication:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(MultiplicationExpr{A: lhs, B: rhs})
		case ruleDivision:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(DivisionExpr{A: lhs, B: rhs})
		case ruleModulo:
			rhs := tokens.Pop()
			lhs := tokens.Pop()

			tokens.Push(ModuloExpr{A: lhs, B: rhs})
		case ruleCall:
			tokens.Push(CallExpr{
				Name:      tokens.Pop().(nameHelper).name,
				Arguments: tokens.GetExpressionList(),
			})
		case ruleName:
			tokens.Push(nameHelper{name:contents})
			
		case ruleList:
			seq := tokens.GetExpressionList()
			tokens.Push(ListExpr{seq})
		
		case ruleNextExpression:
			rhs := tokens.Pop()
			
			list:=tokens.PopExpressionList()
			list.list=append(list.list,rhs)
			tokens.Push(list)
		
		case ruleContents, ruleArguments:
			tokens.SetExpressionList(tokens.PopExpressionList())
			
		case ruleComma:
		case ruleKey:
		case ruleGrouped:
		case ruleLevel0, ruleLevel1, ruleLevel2, ruleLevel3, ruleLevel4:
		case ruleExpression:
		case rulews:
		case rulereq_ws:
		default:
			panic("unhandled:" + rul3s[token.pegRule])
		}
	}

	panic("unreachable")
}

type tokenStack struct {
	list.List

	expressionList *expressionListHelper
}

func (s *tokenStack) Pop() Expression {
	front := s.Front()
	if front == nil {
		return nil
	}

	s.Remove(front)

	return front.Value.(Expression)
}

func (s *tokenStack) Push(expr Expression) {
	s.PushFront(expr)
}

func (s *tokenStack) PopExpressionList() expressionListHelper {
	lhs := s.Pop()
	list, ok:= lhs.(expressionListHelper)
	if !ok {
		list=expressionListHelper{list:[]Expression{lhs}}
	}
	return list
}

func (s *tokenStack) SetExpressionList(list expressionListHelper) {
	s.expressionList = &list
}

func (s *tokenStack) GetExpressionList() []Expression {
	list := s.expressionList
	s.expressionList = nil
	if list==nil {
		return []Expression(nil)
	}
	return list.list
}
