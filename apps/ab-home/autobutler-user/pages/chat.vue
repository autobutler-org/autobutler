<template>
  <div class="h-screen flex flex-col">
    <!-- Chat Header -->
    <div class="bg-white/5 backdrop-blur-lg border-b border-white/10 p-4">
      <div class="max-w-4xl mx-auto flex items-center justify-between">
        <h1 class="text-xl font-semibold text-white">AutoButler Chat</h1>
        <div class="flex items-center space-x-4">
          <button class="text-gray-300 hover:text-white transition-colors">
            <span class="sr-only">Settings</span>
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Chat Messages -->
    <div class="flex-1 overflow-y-auto p-4">
      <div class="max-w-4xl mx-auto space-y-4">
        <div v-for="(message, index) in messages" :key="index" 
             :class="['flex', message.isUser ? 'justify-end' : 'justify-start']">
          <div :class="[
            'max-w-[80%] rounded-lg p-4',
            message.isUser 
              ? 'bg-blue-600 text-white' 
              : 'bg-white/10 text-gray-100'
          ]">
            <p class="whitespace-pre-wrap">{{ message.content }}</p>
            <span class="text-xs mt-2 block opacity-70">
              {{ message.timestamp }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Chat Input -->
    <div class="bg-white/5 backdrop-blur-lg border-t border-white/10 p-4">
      <div class="max-w-4xl mx-auto">
        <form @submit.prevent="sendMessage" class="flex space-x-4">
          <input
            v-model="newMessage"
            type="text"
            placeholder="Type your message..."
            class="flex-1 bg-white/10 border border-white/20 rounded-lg px-4 py-2 text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            @keydown.enter.prevent="sendMessage"
          />
          <button
            type="submit"
            class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg transition-colors duration-200 flex items-center space-x-2"
          >
            <span>Send</span>
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.429A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z" />
            </svg>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const messages = ref([
  {
    content: "Hello! I'm AutoButler, your AI assistant. How can I help you today?",
    isUser: false,
    timestamp: new Date().toLocaleTimeString()
  }
])

const newMessage = ref('')

const sendMessage = () => {
  if (!newMessage.value.trim()) return

  // Add user message
  messages.value.push({
    content: newMessage.value,
    isUser: true,
    timestamp: new Date().toLocaleTimeString()
  })

  // TODO: Implement actual API call to LLM service
  // For now, just echo back a response
  setTimeout(() => {
    messages.value.push({
      content: "I received your message: " + newMessage.value,
      isUser: false,
      timestamp: new Date().toLocaleTimeString()
    })
  }, 1000)

  newMessage.value = ''
}
</script> 