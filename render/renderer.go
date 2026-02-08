package render

import (
	"image"

	"github.com/fogleman/fauxgl"
)

// Renderer wraps FauxGL with TermGL-specific configuration.
// Architecture doc Section 2.3
type Renderer struct {
	width, height int
	context       *fauxgl.Context
}

// NewRenderer creates a renderer at the specified internal resolution.
// For Tier 1 (Sixel), this is pixel resolution.
// For Tier 2 (ANSI), this is sub-cell resolution.
func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		width:   width,
		height:  height,
		context: fauxgl.NewContext(width, height),
	}
}

// RenderFrame produces a single frame as image.NRGBA.
// This is the universal intermediate format consumed by both tiers.
func (r *Renderer) RenderFrame(scene *Scene) *image.NRGBA {
	// Clear buffers
	r.context.ClearColorBufferWith(scene.Ambient)
	r.context.ClearDepthBuffer()

	// Build model-view-projection matrix
	aspect := float64(r.width) / float64(r.height)
	viewMatrix := scene.Camera.ViewMatrix()
	projMatrix := scene.Camera.ProjectionMatrix(aspect)
	vp := projMatrix.Mul(viewMatrix)

	// Render all nodes in the scene graph
	r.renderNode(scene.Root, fauxgl.Identity(), vp, scene.Lights)

	// Type assert image.Image to *image.NRGBA
	img := r.context.Image()
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba
	}

	// If not NRGBA, convert to NRGBA
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, img.At(x, y))
		}
	}
	return nrgba
}

// renderNode recursively renders a node and its children.
func (r *Renderer) renderNode(node *Node, parentTransform fauxgl.Matrix, vp fauxgl.Matrix, lights []Light) {
	if node == nil {
		return
	}

	// Compute world transform for this node
	worldTransform := parentTransform.Mul(node.Transform)

	// If this node has a mesh, render it
	if node.Mesh != nil {
		mvp := vp.Mul(worldTransform)
		shader := NewFlatShader(mvp, lights)
		r.context.Shader = shader
		r.context.DrawMesh(node.Mesh)
	}

	// Recursively render children
	for _, child := range node.Children {
		r.renderNode(child, worldTransform, vp, lights)
	}
}

// RenderFrameWithAux produces a frame plus auxiliary depth/normal buffers.
// Used by Tier 2 for edge-aware character selection.
func (r *Renderer) RenderFrameWithAux(scene *Scene) (*image.NRGBA, *AuxBuffers) {
	// Clear buffers
	r.context.ClearColorBufferWith(scene.Ambient)
	r.context.ClearDepthBuffer()

	// Prepare auxiliary buffers
	aux := &AuxBuffers{
		Width:     r.width,
		Height:    r.height,
		DepthMap:  make([]float64, r.width*r.height),
		NormalMap: make([]fauxgl.Vector, r.width*r.height),
	}

	// Build model-view-projection matrix
	aspect := float64(r.width) / float64(r.height)
	viewMatrix := scene.Camera.ViewMatrix()
	projMatrix := scene.Camera.ProjectionMatrix(aspect)
	vp := projMatrix.Mul(viewMatrix)

	// Render all nodes with auxiliary buffer shader
	r.renderNodeWithAux(scene.Root, fauxgl.Identity(), vp, scene.Lights, aux)

	// Type assert image.Image to *image.NRGBA
	img := r.context.Image()
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba, aux
	}

	// If not NRGBA, convert to NRGBA
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, img.At(x, y))
		}
	}
	return nrgba, aux
}

// renderNodeWithAux recursively renders a node with auxiliary buffers.
func (r *Renderer) renderNodeWithAux(node *Node, parentTransform fauxgl.Matrix, vp fauxgl.Matrix, lights []Light, aux *AuxBuffers) {
	if node == nil {
		return
	}

	// Compute world transform for this node
	worldTransform := parentTransform.Mul(node.Transform)

	// If this node has a mesh, render it with aux shader
	if node.Mesh != nil {
		mvp := vp.Mul(worldTransform)
		shader := NewAuxBufferShader(mvp, lights, r.width, aux.DepthMap, aux.NormalMap)
		r.context.Shader = shader
		r.context.DrawMesh(node.Mesh)
	}

	// Recursively render children
	for _, child := range node.Children {
		r.renderNodeWithAux(child, worldTransform, vp, lights, aux)
	}
}

// Resize changes the internal rendering resolution.
func (r *Renderer) Resize(width, height int) {
	r.width = width
	r.height = height
	r.context = fauxgl.NewContext(width, height)
}
