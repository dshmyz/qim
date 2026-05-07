<template>
  <div class="avatar-cropper-modal" @click="$emit('cancel')">
    <div class="cropper-content" @click.stop>
      <div class="cropper-header">
        <h3>裁剪头像</h3>
        <button class="close-btn" @click="$emit('cancel')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      
      <div class="cropper-body">
        <div class="canvas-container">
          <canvas ref="canvasRef" class="cropper-canvas"></canvas>
        </div>
        
        <div class="preview-section">
          <div class="preview-label">预览</div>
          <div class="preview-sizes">
            <div class="preview-item">
              <canvas ref="previewLargeRef" class="preview-canvas" width="180" height="180"></canvas>
              <span class="preview-size-label">180px</span>
            </div>
            <div class="preview-item">
              <canvas ref="previewMediumRef" class="preview-canvas" width="120" height="120"></canvas>
              <span class="preview-size-label">120px</span>
            </div>
            <div class="preview-item">
              <canvas ref="previewSmallRef" class="preview-canvas" width="64" height="64"></canvas>
              <span class="preview-size-label">64px</span>
            </div>
          </div>
        </div>
        
        <div class="controls-section">
          <div class="control-group">
            <label class="control-label">缩放</label>
            <input 
              type="range" 
              min="0.1" 
              max="3" 
              step="0.1" 
              v-model.number="zoom" 
              class="slider"
              @input="handleZoomChange"
            />
            <span class="control-value">{{ Math.round(zoom * 100) }}%</span>
          </div>
          
          <div class="control-group">
            <label class="control-label">旋转</label>
            <div class="rotate-buttons">
              <button class="rotate-btn" @click="rotate(-90)">
                <i class="fas fa-undo"></i>
              </button>
              <button class="rotate-btn" @click="rotate(90)">
                <i class="fas fa-redo"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
      
      <div class="cropper-footer">
        <button class="btn btn-secondary" @click="$emit('cancel')">取消</button>
        <button class="btn btn-primary" @click="confirmCrop">确认裁剪</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

interface Props {
  imageUrl: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'confirm': [file: File]
  'cancel': []
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const previewLargeRef = ref<HTMLCanvasElement | null>(null)
const previewMediumRef = ref<HTMLCanvasElement | null>(null)
const previewSmallRef = ref<HTMLCanvasElement | null>(null)

const zoom = ref(1)
const rotation = ref(0)
const panX = ref(0)
const panY = ref(0)

let image: HTMLImageElement | null = null
let isDragging = false
let dragStartX = 0
let dragStartY = 0
let dragStartPanX = 0
let dragStartPanY = 0

const CANVAS_SIZE = 400
const CROP_RADIUS = CANVAS_SIZE / 2 - 20

onMounted(() => {
  image = new Image()
  image.crossOrigin = 'anonymous'
  image.onload = () => {
    resetTransform()
    draw()
  }
  image.src = props.imageUrl
  
  const canvas = canvasRef.value
  if (canvas) {
    canvas.addEventListener('mousedown', handleMouseDown)
    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('mouseup', handleMouseUp)
    canvas.addEventListener('mouseleave', handleMouseUp)
    canvas.addEventListener('wheel', handleWheel, { passive: false })
    canvas.addEventListener('touchstart', handleTouchStart, { passive: false })
    canvas.addEventListener('touchmove', handleTouchMove, { passive: false })
    canvas.addEventListener('touchend', handleTouchEnd)
  }
})

onUnmounted(() => {
  const canvas = canvasRef.value
  if (canvas) {
    canvas.removeEventListener('mousedown', handleMouseDown)
    canvas.removeEventListener('mousemove', handleMouseMove)
    canvas.removeEventListener('mouseup', handleMouseUp)
    canvas.removeEventListener('mouseleave', handleMouseUp)
    canvas.removeEventListener('wheel', handleWheel)
    canvas.removeEventListener('touchstart', handleTouchStart)
    canvas.removeEventListener('touchmove', handleTouchMove)
    canvas.removeEventListener('touchend', handleTouchEnd)
  }
})

const resetTransform = () => {
  if (!image) return
  
  const scale = Math.min(CANVAS_SIZE / image.width, CANVAS_SIZE / image.height)
  zoom.value = scale
  rotation.value = 0
  panX.value = 0
  panY.value = 0
}

const draw = () => {
  const canvas = canvasRef.value
  if (!canvas || !image) return
  
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  
  canvas.width = CANVAS_SIZE
  canvas.height = CANVAS_SIZE
  
  ctx.clearRect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
  
  ctx.save()
  ctx.translate(CANVAS_SIZE / 2 + panX.value, CANVAS_SIZE / 2 + panY.value)
  ctx.rotate((rotation.value * Math.PI) / 180)
  ctx.scale(zoom.value, zoom.value)
  ctx.drawImage(image, -image.width / 2, -image.height / 2)
  ctx.restore()
  
  ctx.fillStyle = 'rgba(0, 0, 0, 0.6)'
  ctx.beginPath()
  ctx.rect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
  ctx.arc(CANVAS_SIZE / 2, CANVAS_SIZE / 2, CROP_RADIUS, 0, Math.PI * 2, true)
  ctx.fill()
  
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.arc(CANVAS_SIZE / 2, CANVAS_SIZE / 2, CROP_RADIUS, 0, Math.PI * 2)
  ctx.stroke()
  
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.3)'
  ctx.lineWidth = 1
  const gridSize = CROP_RADIUS / 3
  for (let i = 1; i < 3; i++) {
    ctx.beginPath()
    ctx.moveTo(CANVAS_SIZE / 2 - CROP_RADIUS + i * gridSize, CANVAS_SIZE / 2 - CROP_RADIUS)
    ctx.lineTo(CANVAS_SIZE / 2 - CROP_RADIUS + i * gridSize, CANVAS_SIZE / 2 + CROP_RADIUS)
    ctx.stroke()
    
    ctx.beginPath()
    ctx.moveTo(CANVAS_SIZE / 2 - CROP_RADIUS, CANVAS_SIZE / 2 - CROP_RADIUS + i * gridSize)
    ctx.lineTo(CANVAS_SIZE / 2 + CROP_RADIUS, CANVAS_SIZE / 2 - CROP_RADIUS + i * gridSize)
    ctx.stroke()
  }
  
  updatePreviews()
}

const updatePreviews = () => {
  if (!image) return
  
  const previews = [
    { canvas: previewLargeRef.value, size: 180 },
    { canvas: previewMediumRef.value, size: 120 },
    { canvas: previewSmallRef.value, size: 64 }
  ]
  
  previews.forEach(({ canvas, size }) => {
    if (!canvas) return
    
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    
    canvas.width = size
    canvas.height = size
    
    ctx.clearRect(0, 0, size, size)
    
    const scale = size / (CROP_RADIUS * 2)
    const drawZoom = zoom.value * scale
    
    ctx.save()
    ctx.translate(size / 2, size / 2)
    ctx.rotate((rotation.value * Math.PI) / 180)
    ctx.scale(drawZoom, drawZoom)
    ctx.drawImage(image, -image.width / 2, -image.height / 2)
    ctx.restore()
  })
}

const handleMouseDown = (e: MouseEvent) => {
  isDragging = true
  dragStartX = e.clientX
  dragStartY = e.clientY
  dragStartPanX = panX.value
  dragStartPanY = panY.value
}

const handleMouseMove = (e: MouseEvent) => {
  if (!isDragging) return
  panX.value = dragStartPanX + (e.clientX - dragStartX)
  panY.value = dragStartPanY + (e.clientY - dragStartY)
  draw()
}

const handleMouseUp = () => {
  isDragging = false
}

const handleWheel = (e: WheelEvent) => {
  e.preventDefault()
  const delta = e.deltaY > 0 ? -0.1 : 0.1
  zoom.value = Math.max(0.5, Math.min(3, zoom.value + delta))
  draw()
}

const handleTouchStart = (e: TouchEvent) => {
  e.preventDefault()
  if (e.touches.length === 1) {
    isDragging = true
    dragStartX = e.touches[0].clientX
    dragStartY = e.touches[0].clientY
    dragStartPanX = panX.value
    dragStartPanY = panY.value
  }
}

const handleTouchMove = (e: TouchEvent) => {
  e.preventDefault()
  if (!isDragging || e.touches.length !== 1) return
  panX.value = dragStartPanX + (e.touches[0].clientX - dragStartX)
  panY.value = dragStartPanY + (e.touches[0].clientY - dragStartY)
  draw()
}

const handleTouchEnd = () => {
  isDragging = false
}

const handleZoomChange = () => {
  draw()
}

const rotate = (degrees: number) => {
  rotation.value = (rotation.value + degrees) % 360
  draw()
}

const confirmCrop = () => {
  if (!image) return
  
  const outputCanvas = document.createElement('canvas')
  const outputSize = 512
  outputCanvas.width = outputSize
  outputCanvas.height = outputSize
  
  const ctx = outputCanvas.getContext('2d')
  if (!ctx) return
  
  ctx.clearRect(0, 0, outputSize, outputSize)
  
  const scale = outputSize / (CROP_RADIUS * 2)
  const drawZoom = zoom.value * scale
  
  ctx.save()
  ctx.translate(outputSize / 2, outputSize / 2)
  ctx.rotate((rotation.value * Math.PI) / 180)
  ctx.scale(drawZoom, drawZoom)
  ctx.translate(panX.value * scale, panY.value * scale)
  ctx.drawImage(image, -image.width / 2, -image.height / 2)
  ctx.restore()
  
  ctx.globalCompositeOperation = 'destination-in'
  ctx.beginPath()
  ctx.arc(outputSize / 2, outputSize / 2, outputSize / 2, 0, Math.PI * 2)
  ctx.fill()
  
  outputCanvas.toBlob((blob) => {
    if (blob) {
      const file = new File([blob], 'avatar.png', { type: 'image/png' })
      emit('confirm', file)
    }
  }, 'image/png')
}

watch([zoom, rotation, panX, panY], () => {
  draw()
})
</script>

<style scoped>
.avatar-cropper-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1100;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.cropper-content {
  background: var(--modal-bg, #ffffff);
  border-radius: 16px;
  width: 720px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
  animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.cropper-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.cropper-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color, #111827);
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--hover-color, #f3f4f6);
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-secondary, #6b7280);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--active-color, #e5e7eb);
  color: var(--text-color, #111827);
}

.cropper-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.canvas-container {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

.cropper-canvas {
  border-radius: 12px;
  cursor: move;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.preview-section {
  margin-bottom: 24px;
}

.preview-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary, #6b7280);
  margin-bottom: 12px;
}

.preview-sizes {
  display: flex;
  gap: 24px;
  justify-content: center;
}

.preview-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.preview-canvas {
  border-radius: 50%;
  border: 2px solid var(--border-color, #e5e7eb);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.preview-size-label {
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
}

.controls-section {
  display: flex;
  gap: 32px;
  justify-content: center;
}

.control-group {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.control-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary, #6b7280);
}

.slider {
  width: 200px;
  height: 6px;
  border-radius: 3px;
  background: var(--border-color, #e5e7eb);
  outline: none;
  -webkit-appearance: none;
}

.slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary-color, #3b82f6);
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(59, 130, 246, 0.3);
}

.control-value {
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
  min-width: 40px;
  text-align: center;
}

.rotate-buttons {
  display: flex;
  gap: 8px;
}

.rotate-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: var(--secondary-color, #f3f4f6);
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-color, #374151);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.rotate-btn:hover {
  background: var(--hover-color, #e5e7eb);
  transform: scale(1.05);
}

.cropper-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary {
  background: var(--secondary-color, #f3f4f6);
  color: var(--text-color, #374151);
}

.btn-secondary:hover {
  background: var(--hover-color, #e5e7eb);
}

.btn-primary {
  background: var(--primary-color, #3b82f6);
  color: #ffffff;
}

.btn-primary:hover {
  background: var(--active-color, #2563eb);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

</style>
