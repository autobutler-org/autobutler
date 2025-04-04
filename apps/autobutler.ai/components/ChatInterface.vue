<template>
  <div class="chat-interface">
    <div ref="messagesContainer" class="chat-messages">
      <div
        v-for="message in messages"
        :key="message.id"
        :class="['message', message.role]"
      >
        <div class="message-content">
          <div class="message-header">
            {{ message.role === "user" ? "You" : "AI Assistant" }}
          </div>
          <div class="message-text">{{ message.content }}</div>
          <div class="message-timestamp">
            {{ formatTime(message.timestamp) }}
          </div>
        </div>
      </div>
    </div>

    <div class="chat-input">
      <textarea
        v-model="userInput"
        placeholder="Type your message here..."
        rows="3"
        @keydown.enter.prevent="sendMessage"
      ></textarea>
      <button
        :disabled="!userInput.trim()"
        class="send-button"
        @click="sendMessage"
      >
        Send
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue";

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
}

const messages = ref<Message[]>([]);
const userInput = ref("");
const messagesContainer = ref<HTMLElement | null>(null);

const formatTime = (date: Date) =>
  new Intl.DateTimeFormat("en-US", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
};

const sendMessage = async () => {
  if (!userInput.value.trim()) return;

  // Add user message
  messages.value.push({
    id: crypto.randomUUID(),
    role: "user",
    content: userInput.value,
    timestamp: new Date(),
  });

  userInput.value = "";
  const userMessage = userInput.value;
  console.log("User message:", userMessage);

  // Simulate AI response (replace with actual API call later)
  try {
    // Placeholder for API call
    // const response = await callLLMAPI(userMessage);

    // Simulate API delay
    await new Promise((resolve) => setTimeout(resolve, 1000));

    messages.value.push({
      id: crypto.randomUUID(),
      role: "assistant",
      content:
        "This is a placeholder response. Replace with actual LLM integration.",
      timestamp: new Date(),
    });
  } catch (error) {
    console.error("Failed to get AI response:", error);
  }
};

// Auto-scroll when new messages arrive
watch(() => messages.value.length, scrollToBottom);

onMounted(scrollToBottom);
</script>

<style scoped>
.chat-interface {
  display: flex;
  flex-direction: column;
  height: 70vh;
  background: rgba(28, 32, 34, 0.95);
  border-radius: 8px;
  overflow: hidden;
}

.chat-messages {
  flex-grow: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.message {
  max-width: 80%;
  padding: 1rem;
  border-radius: 8px;
  animation: fadeIn 0.3s ease;
}

.message.user {
  align-self: flex-end;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.15),
    rgba(0, 187, 255, 0.15)
  );
  border: 1px solid rgba(0, 255, 170, 0.2);
}

.message.assistant {
  align-self: flex-start;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.message-header {
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.7);
  margin-bottom: 0.5rem;
}

.message-text {
  color: #fff;
  line-height: 1.5;
}

.message-timestamp {
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.5);
  margin-top: 0.5rem;
  text-align: right;
}

.chat-input {
  padding: 1rem;
  background: rgba(255, 255, 255, 0.05);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  gap: 1rem;
}

textarea {
  flex-grow: 1;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  color: #fff;
  padding: 0.8rem;
  resize: none;
  transition: all 0.3s ease;
}

textarea:focus {
  outline: none;
  border-color: rgba(0, 255, 170, 0.3);
  background: rgba(255, 255, 255, 0.15);
}

.send-button {
  padding: 0 1.5rem;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.2),
    rgba(0, 187, 255, 0.2)
  );
  border: 1px solid rgba(0, 255, 170, 0.3);
  border-radius: 4px;
  color: #fff;
  cursor: pointer;
  transition: all 0.3s ease;
}

.send-button:hover:not(:disabled) {
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.3),
    rgba(0, 187, 255, 0.3)
  );
}

.send-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Custom scrollbar */
.chat-messages::-webkit-scrollbar {
  width: 8px;
}

.chat-messages::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.1);
}

.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}
</style>
