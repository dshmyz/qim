<template>
  <!-- 小程序处理组件 -->
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
// 小程序处理组件，用于处理小程序相关的逻辑

// 显示小程序模态框
export const showMiniAppModal = (miniApp: any) => {
  // 创建小程序模态框
  const modal = document.createElement('div')
  modal.className = 'mini-app-modal'
  
  // 根据小程序ID显示不同内容
  let content = ''
  if (miniApp.id === 'calculator' || miniApp.id === '1') {
    // 计算器小程序
    content = `
      <div class="mini-app-modal-content calculator-app">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="calculator-container">
            <div class="calculator-display">
              <div class="calculator-result" id="calculator-result">0</div>
              <div class="calculator-input" id="calculator-input"></div>
            </div>
            <div class="calculator-buttons">
              <div class="calculator-row">
                <button class="calculator-btn calculator-btn-clear" data-value="C">C</button>
                <button class="calculator-btn calculator-btn-operator" data-value="backspace">←</button>
                <button class="calculator-btn calculator-btn-operator" data-value="%">%</button>
                <button class="calculator-btn calculator-btn-operator" data-value="/">÷</button>
              </div>
              <div class="calculator-row">
                <button class="calculator-btn calculator-btn-number" data-value="7">7</button>
                <button class="calculator-btn calculator-btn-number" data-value="8">8</button>
                <button class="calculator-btn calculator-btn-number" data-value="9">9</button>
                <button class="calculator-btn calculator-btn-operator" data-value="*">×</button>
              </div>
              <div class="calculator-row">
                <button class="calculator-btn calculator-btn-number" data-value="4">4</button>
                <button class="calculator-btn calculator-btn-number" data-value="5">5</button>
                <button class="calculator-btn calculator-btn-number" data-value="6">6</button>
                <button class="calculator-btn calculator-btn-operator" data-value="-">-</button>
              </div>
              <div class="calculator-row">
                <button class="calculator-btn calculator-btn-number" data-value="1">1</button>
                <button class="calculator-btn calculator-btn-number" data-value="2">2</button>
                <button class="calculator-btn calculator-btn-number" data-value="3">3</button>
                <button class="calculator-btn calculator-btn-operator" data-value="+">+</button>
              </div>
              <div class="calculator-row">
                <button class="calculator-btn calculator-btn-number" data-value="0">0</button>
                <button class="calculator-btn calculator-btn-number" data-value=".">.</button>
                <button class="calculator-btn calculator-btn-equals" data-value="=">=</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    `
  } else if (miniApp.id === 'notepad') {
    // 记事本小程序
    content = `
      <div class="mini-app-modal-content notepad-app">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="notepad-container">
            <input type="text" id="notepad-title" class="notepad-title" placeholder="输入标题" />
            <textarea id="notepad-content" class="notepad-content" placeholder="输入内容"></textarea>
            <div class="notepad-actions">
              <button id="notepad-save" class="notepad-btn">保存</button>
              <button id="notepad-clear" class="notepad-btn">清空</button>
            </div>
          </div>
        </div>
      </div>
    `
  } else if (miniApp.id === 'password-generator') {
    // 密码生成器小程序
    content = `
      <div class="mini-app-modal-content password-generator-app">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="password-generator-container">
            <div class="password-result">
              <input type="text" id="password-result" class="password-result-input" readonly />
              <button id="password-copy" class="password-copy-btn">复制</button>
            </div>
            <div class="password-options">
              <div class="password-option">
                <label>密码长度</label>
                <input type="range" id="password-length" min="8" max="32" value="16" />
                <span id="password-length-value">16</span>
              </div>
              <div class="password-option">
                <label><input type="checkbox" id="include-uppercase" checked /> 包含大写字母</label>
              </div>
              <div class="password-option">
                <label><input type="checkbox" id="include-lowercase" checked /> 包含小写字母</label>
              </div>
              <div class="password-option">
                <label><input type="checkbox" id="include-numbers" checked /> 包含数字</label>
              </div>
              <div class="password-option">
                <label><input type="checkbox" id="include-symbols" checked /> 包含特殊字符</label>
              </div>
            </div>
            <button id="generate-password" class="generate-btn">生成密码</button>
          </div>
        </div>
      </div>
    `
  } else if (miniApp.id === 'todo') {
    // 待办事项小程序
    content = `
      <div class="mini-app-modal-content todo-app">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="todo-container">
            <div class="todo-input-container">
              <input type="text" id="todo-input" class="todo-input" placeholder="输入新任务" />
              <button id="add-todo" class="add-todo-btn">添加</button>
            </div>
            <div id="todo-list" class="todo-list">
              <!-- 任务列表将在这里动态生成 -->
            </div>
          </div>
        </div>
      </div>
    `
  } else if (miniApp.id === 'short-link') {
    // 短链接生成器小程序
    content = `
      <div class="mini-app-modal-content short-link-app">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="short-link-container">
            <div class="short-link-input-section">
              <label>原始URL</label>
              <textarea id="short-link-input" class="original-url-input" placeholder="请输入要缩短的URL" rows="3"></textarea>
            </div>
            <button id="generate-short-link" class="generate-btn">生成短链接</button>
            <div id="short-link-result" class="short-link-result" style="display: none;">
              <label>生成的短链接</label>
              <div class="short-link-output">
                <input type="text" id="short-link-output-input" class="short-url-input" readonly />
                <button id="copy-short-link" class="copy-btn">复制</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    `
  } else {
    // 其他小程序
    content = `
      <div class="mini-app-modal-content">
        <div class="mini-app-modal-header">
          <div class="mini-app-modal-title">${miniApp.name}</div>
          <button class="mini-app-modal-close">×</button>
        </div>
        <div class="mini-app-modal-body">
          <div class="mini-app-modal-icon">
            <img src="${miniApp.icon}" alt="${miniApp.name}" />
          </div>
          <div class="mini-app-modal-description">${miniApp.description}</div>
          <div class="mini-app-modal-content-area">
            <p>小程序功能正在开发中...</p>
          </div>
        </div>
      </div>
    `
  }
  
  modal.innerHTML = content
  document.body.appendChild(modal)
  
  // 关闭按钮事件
  const closeBtn = modal.querySelector('.mini-app-modal-close')
  if (closeBtn) {
    closeBtn.addEventListener('click', () => {
      document.body.removeChild(modal)
    })
  }
  
  // 点击模态框外部关闭
  modal.addEventListener('click', (e) => {
    if (e.target === modal) {
      document.body.removeChild(modal)
    }
  })
  
  // 计算器功能
  if (miniApp.id === 'calculator' || miniApp.id === '1') {
    const resultElement = document.getElementById('calculator-result')
    const inputElement = document.getElementById('calculator-input')
    let result = '0'
    let input = ''
    let operator = ''
    let firstNumber = ''
    let secondNumber = ''
    let shouldReset = false
    
    const updateDisplay = () => {
      if (resultElement) resultElement.textContent = result
      if (inputElement) inputElement.textContent = input
    }
    
    const calculate = () => {
      if (!firstNumber || !secondNumber || !operator) return
      
      const num1 = parseFloat(firstNumber)
      const num2 = parseFloat(secondNumber)
      let calcResult = 0
      
      switch (operator) {
        case '+':
          calcResult = num1 + num2
          break
        case '-':
          calcResult = num1 - num2
          break
        case '*':
          calcResult = num1 * num2
          break
        case '/':
          calcResult = num1 / num2
          break
        case '%':
          calcResult = num1 % num2
          break
      }
      
      result = calcResult.toString()
      input = `${firstNumber} ${operator} ${secondNumber} = ${result}`
      firstNumber = result
      secondNumber = ''
      operator = ''
      shouldReset = true
      updateDisplay()
    }
    
    // 绑定按钮事件
    const buttons = modal.querySelectorAll('.calculator-btn')
    buttons.forEach(button => {
      button.addEventListener('click', () => {
        const value = button.getAttribute('data-value')
        if (!value) return
        
        if (value === 'C') {
          // 清除所有
          result = '0'
          input = ''
          firstNumber = ''
          secondNumber = ''
          operator = ''
          shouldReset = false
        } else if (value === 'backspace') {
          // 退格
          if (shouldReset) {
            result = '0'
            shouldReset = false
          } else if (result !== '0') {
            result = result.slice(0, -1) || '0'
          }
        } else if (['+', '-', '*', '/', '%'].includes(value)) {
          // 运算符
          if (firstNumber === '') {
            firstNumber = result
          } else if (secondNumber !== '') {
            calculate()
            return
          }
          operator = value
          input = `${firstNumber} ${operator}`
          shouldReset = true
        } else if (value === '=') {
          // 等于
          if (firstNumber && operator) {
            secondNumber = result
            calculate()
          }
        } else {
          // 数字或小数点
          if (shouldReset) {
            result = value
            shouldReset = false
          } else {
            if (value === '.' && result.includes('.')) return
            if (result === '0' && value !== '.') {
              result = value
            } else {
              result += value
            }
          }
        }
        
        updateDisplay()
      })
    })
  } else if (miniApp.id === 'notepad') {
    // 记事本功能
    const titleInput = document.getElementById('notepad-title') as HTMLInputElement
    const contentInput = document.getElementById('notepad-content') as HTMLTextAreaElement
    const saveBtn = document.getElementById('notepad-save')
    const clearBtn = document.getElementById('notepad-clear')
    
    // 加载保存的笔记
    const savedNote = localStorage.getItem('notepad-note')
    if (savedNote) {
      try {
        const note = JSON.parse(savedNote)
        if (titleInput) titleInput.value = note.title || ''
        if (contentInput) contentInput.value = note.content || ''
      } catch (e) {
        console.error('解析笔记失败:', e)
      }
    }
    
    // 保存笔记
    if (saveBtn) {
      saveBtn.addEventListener('click', () => {
        const note = {
          title: titleInput?.value || '',
          content: contentInput?.value || '',
          timestamp: Date.now()
        }
        localStorage.setItem('notepad-note', JSON.stringify(note))
        ElMessage.success('笔记保存成功')
      })
    }
    
    // 清空笔记
    if (clearBtn) {
      clearBtn.addEventListener('click', () => {
        if (confirm('确定要清空笔记吗？')) {
          if (titleInput) titleInput.value = ''
          if (contentInput) contentInput.value = ''
          localStorage.removeItem('notepad-note')
          ElMessage.success('笔记已清空')
        }
      })
    }
  } else if (miniApp.id === 'password-generator') {
    // 密码生成器功能
    const resultInput = document.getElementById('password-result') as HTMLInputElement
    const copyBtn = document.getElementById('password-copy')
    const lengthInput = document.getElementById('password-length') as HTMLInputElement
    const lengthValue = document.getElementById('password-length-value')
    const uppercaseCheck = document.getElementById('include-uppercase') as HTMLInputElement
    const lowercaseCheck = document.getElementById('include-lowercase') as HTMLInputElement
    const numbersCheck = document.getElementById('include-numbers') as HTMLInputElement
    const symbolsCheck = document.getElementById('include-symbols') as HTMLInputElement
    const generateBtn = document.getElementById('generate-password')
    
    // 更新长度显示
    if (lengthInput && lengthValue) {
      lengthInput.addEventListener('input', () => {
        lengthValue.textContent = lengthInput.value
      })
    }
    
    // 生成密码
    const generatePassword = () => {
      const length = parseInt(lengthInput?.value || '16')
      const includeUppercase = uppercaseCheck?.checked || false
      const includeLowercase = lowercaseCheck?.checked || false
      const includeNumbers = numbersCheck?.checked || false
      const includeSymbols = symbolsCheck?.checked || false
      
      let charset = ''
      if (includeUppercase) charset += 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
      if (includeLowercase) charset += 'abcdefghijklmnopqrstuvwxyz'
      if (includeNumbers) charset += '0123456789'
      if (includeSymbols) charset += '!@#$%^&*()_+[]{}|;:,.<>?'
      
      if (charset === '') {
        ElMessage.warning('请至少选择一种字符类型')
        return
      }
      
      let password = ''
      for (let i = 0; i < length; i++) {
        const randomIndex = Math.floor(Math.random() * charset.length)
        password += charset[randomIndex]
      }
      
      if (resultInput) {
        resultInput.value = password
      }
    }
    
    // 绑定生成按钮
    if (generateBtn) {
      generateBtn.addEventListener('click', generatePassword)
    }
    
    // 复制密码
    if (copyBtn) {
      copyBtn.addEventListener('click', () => {
        if (resultInput && resultInput.value) {
          navigator.clipboard.writeText(resultInput.value)
            .then(() => {
              ElMessage.success('密码已复制到剪贴板')
            })
            .catch(err => {
              console.error('复制失败:', err)
              ElMessage.error('复制失败，请手动复制')
            })
        }
      })
    }
    
    // 初始生成密码
    generatePassword()
  } else if (miniApp.id === 'todo') {
    // 待办事项功能
    const todoInput = document.getElementById('todo-input') as HTMLInputElement
    const addBtn = document.getElementById('add-todo')
    const todoList = document.getElementById('todo-list')
    
    // 加载保存的待办事项
    const loadTodos = () => {
      const savedTodos = localStorage.getItem('todo-list')
      if (savedTodos) {
        try {
          const todos = JSON.parse(savedTodos)
          renderTodos(todos)
        } catch (e) {
          console.error('解析待办事项失败:', e)
        }
      }
    }
    
    // 渲染待办事项
    const renderTodos = (todos: any[]) => {
      if (!todoList) return
      todoList.innerHTML = ''
      todos.forEach((todo, index) => {
        const todoItem = document.createElement('div')
        todoItem.className = `todo-item ${todo.completed ? 'completed' : ''}`
        todoItem.innerHTML = `
          <input type="checkbox" ${todo.completed ? 'checked' : ''} data-index="${index}">
          <span class="todo-text">${todo.text}</span>
          <button class="todo-delete" data-index="${index}">×</button>
        `
        todoList.appendChild(todoItem)
      })
      
      // 绑定事件
      const checkboxes = todoList.querySelectorAll('input[type="checkbox"]')
      checkboxes.forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
          const index = parseInt((e.target as HTMLInputElement).getAttribute('data-index') || '0')
          toggleTodo(index)
        })
      })
      
      const deleteBtns = todoList.querySelectorAll('.todo-delete')
      deleteBtns.forEach(btn => {
        btn.addEventListener('click', (e) => {
          const index = parseInt((e.target as HTMLButtonElement).getAttribute('data-index') || '0')
          deleteTodo(index)
        })
      })
    }
    
    // 添加待办事项
    const addTodo = () => {
      if (!todoInput || !todoInput.value.trim()) return
      
      const savedTodos = localStorage.getItem('todo-list')
      const todos = savedTodos ? JSON.parse(savedTodos) : []
      
      todos.push({
        text: todoInput.value.trim(),
        completed: false
      })
      
      localStorage.setItem('todo-list', JSON.stringify(todos))
      renderTodos(todos)
      todoInput.value = ''
    }
    
    // 切换待办事项状态
    const toggleTodo = (index: number) => {
      const savedTodos = localStorage.getItem('todo-list')
      if (!savedTodos) return
      
      const todos = JSON.parse(savedTodos)
      if (index >= 0 && index < todos.length) {
        todos[index].completed = !todos[index].completed
        localStorage.setItem('todo-list', JSON.stringify(todos))
        renderTodos(todos)
      }
    }
    
    // 删除待办事项
    const deleteTodo = (index: number) => {
      const savedTodos = localStorage.getItem('todo-list')
      if (!savedTodos) return
      
      const todos = JSON.parse(savedTodos)
      if (index >= 0 && index < todos.length) {
        todos.splice(index, 1)
        localStorage.setItem('todo-list', JSON.stringify(todos))
        renderTodos(todos)
      }
    }
    
    // 绑定添加按钮
    if (addBtn) {
      addBtn.addEventListener('click', addTodo)
    }
    
    // 绑定回车键
    if (todoInput) {
      todoInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
          addTodo()
        }
      })
    }
    
    // 初始加载
    loadTodos()
  } else if (miniApp.id === 'short-link') {
    // 短链接生成器功能
    const shortLinkInput = document.getElementById('short-link-input') as HTMLTextAreaElement
    const generateBtn = document.getElementById('generate-short-link')
    const resultDiv = document.getElementById('short-link-result')
    const outputInput = document.getElementById('short-link-output-input') as HTMLInputElement
    const copyBtn = document.getElementById('copy-short-link')
    const serverUrl = localStorage.getItem('serverUrl') || 'https://qim.buaa.edu.cn'

    // 生成短链接
    const generateShortLink = async () => {
      const url = shortLinkInput?.value.trim()
      if (!url) {
        ElMessage.warning('请输入要缩短的URL')
        return
      }

      try {
        const token = localStorage.getItem('token')
        const response = await fetch(`${serverUrl}/api/v1/shortlinks`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ original_url: url })
        })

        if (!response.ok) {
          throw new Error('生成短链接失败')
        }

        const data = await response.json()
        if (data.code === 0 && data.data) {
          if (outputInput) {
            outputInput.value = data.data.short_url
          }
          if (resultDiv) {
            resultDiv.style.display = 'block'
          }
          ElMessage.success('短链接生成成功')
        }
      } catch (error) {
        console.error('生成短链接失败:', error)
        ElMessage.error('生成短链接失败')
      }
    }

    // 复制短链接
    const copyShortLink = async () => {
      if (outputInput && outputInput.value) {
        try {
          await navigator.clipboard.writeText(outputInput.value)
          ElMessage.success('短链接已复制到剪贴板')
        } catch (error) {
          console.error('复制失败:', error)
          ElMessage.error('复制失败，请手动复制')
        }
      }
    }

    // 绑定生成按钮
    if (generateBtn) {
      generateBtn.addEventListener('click', generateShortLink)
    }

    // 绑定复制按钮
    if (copyBtn) {
      copyBtn.addEventListener('click', copyShortLink)
    }
  }
}

// 打开小程序
export const openMiniApp = (miniAppData: any) => {
  console.log('打开小程序:', miniAppData)
  // 这里实现打开小程序的逻辑
  if (miniAppData) {
    console.log(`打开${miniAppData.name}小程序`)
    // 调用showMiniAppModal函数打开小程序
    showMiniAppModal(miniAppData)
  }
}
</script>

<style scoped>
/* 小程序模态框样式 */
.mini-app-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.mini-app-modal-content {
  background: var(--sidebar-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
.mini-app-modal-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.mini-app-modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}
.mini-app-modal-close {
  background: none;
  border: none;
  font-size: 20px;
  color: var(--text-secondary);
  cursor: pointer;
}
.mini-app-modal-body {
  padding: 20px;
}
.mini-app-modal-icon {
  text-align: center;
  margin-bottom: 16px;
}
.mini-app-modal-icon img {
  width: 80px;
  height: 80px;
  border-radius: 16px;
}
.mini-app-modal-description {
  text-align: center;
  color: var(--text-secondary);
  margin-bottom: 20px;
}
.mini-app-modal-content-area {
  background: var(--content-bg);
  padding: 16px;
  border-radius: 8px;
}
.mini-app-modal-content-area h3 {
  margin-top: 0;
  color: var(--text-color);
}
.mini-app-modal-content-area p {
  color: var(--text-secondary);
  line-height: 1.5;
}
.mini-app-modal-footer {
  padding: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
}
.mini-app-modal-btn {
  padding: 8px 16px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

/* 计算器样式 */
.calculator-app {
  max-width: 350px;
}
.calculator-container {
  width: 100%;
}
.calculator-display {
  background: var(--content-bg);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}
.calculator-result {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
  text-align: right;
  margin-bottom: 8px;
}
.calculator-input {
  font-size: 14px;
  color: var(--text-secondary);
  text-align: right;
  min-height: 18px;
}
.calculator-buttons {
  display: grid;
  grid-gap: 8px;
}
.calculator-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-gap: 8px;
}
.calculator-btn {
  padding: 16px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}
.calculator-btn:hover {
  opacity: 0.8;
}
.calculator-btn-number {
  background: var(--content-bg);
  color: var(--text-color);
}
.calculator-btn-operator {
  background: var(--primary-color);
  color: white;
}
.calculator-btn-clear {
  background: #ff6b6b;
  color: white;
}
.calculator-btn-equals {
  background: #4ecdc4;
  color: white;
  grid-column: span 2;
}

/* 记事本样式 */
.notepad-app {
  max-width: 500px;
}
.notepad-container {
  width: 100%;
}
.notepad-title {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  margin-bottom: 10px;
  font-size: 16px;
  font-weight: 600;
  background: var(--content-bg);
  color: var(--text-color);
}
.notepad-content {
  width: 100%;
  height: 200px;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  margin-bottom: 10px;
  resize: vertical;
  background: var(--content-bg);
  color: var(--text-color);
  font-family: inherit;
}
.notepad-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
}
.notepad-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}
.notepad-btn:first-child {
  background: var(--primary-color);
  color: white;
}
.notepad-btn:last-child {
  background: var(--content-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

/* 密码生成器样式 */
.password-generator-app {
  max-width: 400px;
}
.password-result {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}
.password-result-input {
  flex: 1;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--content-bg);
  color: var(--text-color);
}
.password-copy-btn {
  padding: 0 16px;
  border: 1px solid var(--primary-color);
  border-radius: 4px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}
.password-options {
  margin-bottom: 20px;
}
.password-option {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.password-option label {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--text-color);
}
.password-option input[type="range"] {
  flex: 1;
}
.password-option span {
  width: 30px;
  text-align: right;
  color: var(--text-secondary);
}
.generate-btn {
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: 4px;
  background: var(--primary-color);
  color: white;
  font-size: 16px;
  cursor: pointer;
}

/* 待办事项样式 */
.todo-app {
  max-width: 400px;
}
.todo-input-container {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}
.todo-input {
  flex: 1;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--content-bg);
  color: var(--text-color);
}
.add-todo-btn {
  padding: 0 16px;
  border: none;
  border-radius: 4px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}
.todo-list {
  max-height: 250px;
  overflow-y: auto;
}
.todo-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-bottom: 1px solid var(--border-color);
}
.todo-item.completed .todo-text {
  text-decoration: line-through;
  color: var(--text-secondary);
}
.todo-text {
  flex: 1;
  color: var(--text-color);
}
.todo-delete {
  background: none;
  border: none;
  color: #ff6b6b;
  cursor: pointer;
  font-size: 16px;
}
</style>