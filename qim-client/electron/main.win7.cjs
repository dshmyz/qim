const path = require('path')
const { pathToFileURL } = require('url')

const mainUrl = pathToFileURL(path.join(__dirname, 'main.js')).href

import(mainUrl).catch((error) => {
  console.error('Failed to load Electron main process:', error)
  throw error
})
