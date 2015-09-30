package dom

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/neelance/dom/domutil"
)

type Markup interface {
	Apply(element *Element)
}

type Instance interface {
	Node() *js.Object
}

type Spec interface {
	Reconcile(oldSpec Spec)
	Markup
	Instance
}

func Render(spec Spec, container *js.Object) {
	spec.Reconcile(nil)
	container.Call("appendChild", spec.Node())
}

func RenderAsBody(spec Spec) {
	body := js.Global.Get("document").Call("createElement", "body")
	Render(spec, body)
	js.Global.Get("document").Set("body", body)
}

type TextSpec struct {
	text string
	node *js.Object
}

func Text(text string) *TextSpec {
	return &TextSpec{text: text}
}

func (s *TextSpec) Apply(element *Element) {
	element.Children = append(element.Children, s)
}

func (s *TextSpec) Reconcile(oldSpec Spec) {
	if oldText, ok := oldSpec.(*TextSpec); ok {
		s.node = oldText.node
		if oldText.text != s.text {
			s.node.Set("nodeValue", s.text)
		}
		return
	}

	s.node = js.Global.Get("document").Call("createTextNode", s.text)
}

func (s *TextSpec) Node() *js.Object {
	return s.node
}

type Element struct {
	TagName        string
	Properties     []*Property
	Styles         []*Style
	EventListeners []*EventListener
	Children       []Spec
	node           *js.Object
}

func (e *Element) AddChild(s Spec) {
	e.Children = append(e.Children, s)
}

func (e *Element) Apply(element *Element) {
	element.Children = append(element.Children, e)
}

func (e *Element) Reconcile(oldSpec Spec) {
	if oldElement, ok := oldSpec.(*Element); ok && oldElement.TagName == e.TagName {
		e.node = oldElement.node
		// TODO fix updating
		for _, p := range oldElement.Properties {
			e.node.Set(p.Name, nil)
		}
		for _, p := range e.Properties {
			e.node.Set(p.Name, p.Value)
		}
		style := e.node.Get("style")
		for _, s := range e.Styles {
			style.Call("setProperty", s.Name, s.Value)
		}
		// TODO better list element reuse
		for i, newChild := range e.Children {
			if i >= len(oldElement.Children) {
				newChild.Reconcile(nil)
				e.node.Call("appendChild", newChild.Node())
				continue
			}
			oldChild := oldElement.Children[i]
			newChild.Reconcile(oldChild)
			domutil.ReplaceNode(newChild.Node(), oldChild.Node())
		}
		for i := len(e.Children); i < len(oldElement.Children); i++ {
			domutil.RemoveNode(oldElement.Children[i].Node())
		}
		return
	}

	e.node = js.Global.Get("document").Call("createElement", e.TagName)
	for _, p := range e.Properties {
		e.node.Set(p.Name, p.Value)
	}
	style := e.node.Get("style")
	for _, s := range e.Styles {
		style.Call("setProperty", s.Name, s.Value)
	}
	for _, l := range e.EventListeners {
		e.node.Call("addEventListener", l.Name, func(jsEvent *js.Object) {
			if l.CallPreventDefault {
				jsEvent.Call("preventDefault")
			}
			l.Listener(&Event{Target: jsEvent.Get("target")})
		})
	}
	for _, c := range e.Children {
		c.Reconcile(nil)
		e.node.Call("appendChild", c.Node())
	}
}

func (e *Element) Node() *js.Object {
	return e.node
}

type Property struct {
	Name  string
	Value interface{}
}

func (p *Property) Apply(element *Element) {
	element.Properties = append(element.Properties, p)
}

type Style struct {
	Name  string
	Value interface{}
}

func (p *Style) Apply(element *Element) {
	element.Styles = append(element.Styles, p)
}

type EventListener struct {
	Name               string
	Listener           func(*Event)
	CallPreventDefault bool
}

func (l *EventListener) PreventDefault() *EventListener {
	l.CallPreventDefault = true
	return l
}

func (l *EventListener) Apply(element *Element) {
	element.EventListeners = append(element.EventListeners, l)
}

type Event struct {
	Target *js.Object
}

// SetTitle sets the title of the document.
func SetTitle(title string) {
	js.Global.Get("document").Set("title", title)
}

// AddStylesheed adds an external stylesheet to the document.
func AddStylesheet(url string) {
	link := js.Global.Get("document").Call("createElement", "link")
	link.Set("rel", "stylesheet")
	link.Set("href", url)
	js.Global.Get("document").Get("head").Call("appendChild", link)
}

type List []Markup

func (g List) Apply(element *Element) {
	for _, m := range g {
		if m != nil {
			m.Apply(element)
		}
	}
}

func If(cond bool, markup ...Markup) Markup {
	if cond {
		return List(markup)
	}
	return nil
}
