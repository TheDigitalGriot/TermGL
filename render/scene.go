package render

import (
	"time"

	"github.com/fogleman/fauxgl"
)

// Scene is the top-level container shared by both output tiers.
// Architecture doc Section 2.4
type Scene struct {
	Root    *Node
	Camera  Camera
	Lights  []Light
	Ambient fauxgl.Color
}

// NewScene creates a new scene with default settings.
func NewScene() *Scene {
	return &Scene{
		Root:    NewNode(),
		Camera:  NewCamera(),
		Lights:  []Light{},
		Ambient: fauxgl.Black,
	}
}

// Tick advances animation state.
// Integrates with Harmonica spring physics for smooth interpolation.
func (s *Scene) Tick(t time.Time) {
	// TODO: Integrate with Harmonica spring physics
	// For now, this is a placeholder that can be extended
	// to update Position, Rotation, Scale on nodes based on time
	s.tickNode(s.Root, t)
}

// tickNode recursively updates a node and its children.
func (s *Scene) tickNode(node *Node, t time.Time) {
	if node == nil {
		return
	}

	// Update transform matrix from Position, Rotation, Scale
	node.updateTransform()

	// Recursively tick children
	for _, child := range node.Children {
		s.tickNode(child, t)
	}
}

// Node represents a transform hierarchy element.
// Parent-child relationships with accumulated matrices.
// Architecture doc Section 2.4
type Node struct {
	Mesh      *fauxgl.Mesh // nil for group nodes
	Transform fauxgl.Matrix
	Children  []*Node

	// Animation targets (driven by Harmonica springs)
	Position fauxgl.Vector
	Rotation fauxgl.Vector // Euler angles in radians
	Scale    fauxgl.Vector
}

// NewNode creates a new node with identity transform.
func NewNode() *Node {
	return &Node{
		Transform: fauxgl.Identity(),
		Children:  []*Node{},
		Position:  fauxgl.V(0, 0, 0),
		Rotation:  fauxgl.V(0, 0, 0),
		Scale:     fauxgl.V(1, 1, 1),
	}
}

// NewMeshNode creates a new node with a mesh.
func NewMeshNode(mesh *fauxgl.Mesh) *Node {
	node := NewNode()
	node.Mesh = mesh
	return node
}

// AddChild adds a child node under this node.
func (n *Node) AddChild(child *Node) {
	n.Children = append(n.Children, child)
}

// updateTransform builds the transform matrix from Position, Rotation, Scale.
func (n *Node) updateTransform() {
	// Build transformation matrix from TRS (Translation, Rotation, Scale)
	// Order: Scale -> Rotate -> Translate
	t := fauxgl.Translate(n.Position)
	rx := fauxgl.Rotate(fauxgl.V(1, 0, 0), n.Rotation.X)
	ry := fauxgl.Rotate(fauxgl.V(0, 1, 0), n.Rotation.Y)
	rz := fauxgl.Rotate(fauxgl.V(0, 0, 1), n.Rotation.Z)
	s := fauxgl.Scale(n.Scale)

	// Combine: T * Rz * Ry * Rx * S
	n.Transform = t.Mul(rz).Mul(ry).Mul(rx).Mul(s)
}

// WorldTransform computes accumulated transform from root to this node.
// Note: This requires parent tracking, which we don't have in the current structure.
// For now, the renderer handles transform accumulation during traversal.
func (n *Node) WorldTransform() fauxgl.Matrix {
	// This would require parent pointer to walk up the tree
	// For now, return the local transform
	// The renderer accumulates transforms during traversal in renderNode
	return n.Transform
}

// SetPosition sets the position and updates the transform.
func (n *Node) SetPosition(x, y, z float64) {
	n.Position = fauxgl.V(x, y, z)
	n.updateTransform()
}

// SetRotation sets the rotation (Euler angles in radians) and updates the transform.
func (n *Node) SetRotation(x, y, z float64) {
	n.Rotation = fauxgl.V(x, y, z)
	n.updateTransform()
}

// SetScale sets the scale and updates the transform.
func (n *Node) SetScale(x, y, z float64) {
	n.Scale = fauxgl.V(x, y, z)
	n.updateTransform()
}
