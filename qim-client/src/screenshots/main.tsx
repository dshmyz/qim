import { createRoot } from 'react-dom/client'
import ScreenshotApp from './ScreenshotApp'

const root = createRoot(document.getElementById('screenshot-root')!)
root.render(<ScreenshotApp />)