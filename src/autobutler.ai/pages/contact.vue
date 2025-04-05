<template>
  <main class="contact-page">
    <div class="contact-content">
      <h1>Contact Us</h1>
      <div class="contact-info">
        <div class="section">
          <h2>Get in Touch</h2>
          <p>Have questions or need assistance? We're here to help!</p>
          <div class="contact-details">
            <p><strong>Email:</strong> support@autobutler.ai</p>
            <p>
              <strong>Hours:</strong> Monday - Friday, 9:00 AM - 5:00 PM EST
            </p>
          </div>
        </div>

        <form class="contact-form" @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="name">Name</label>
            <input
              id="name"
              v-model="formData.name"
              type="text"
              required
              placeholder="Your name"
            />
          </div>

          <div class="form-group">
            <label for="email">Email</label>
            <input
              id="email"
              v-model="formData.email"
              type="email"
              required
              placeholder="your@email.com"
            />
          </div>

          <div class="form-group">
            <label for="subject">Subject</label>
            <input
              id="subject"
              v-model="formData.subject"
              type="text"
              required
              placeholder="How can we help?"
            />
          </div>

          <div class="form-group">
            <label for="message">Message</label>
            <textarea
              id="message"
              v-model="formData.message"
              required
              rows="5"
              placeholder="Your message..."
            ></textarea>
          </div>

          <button type="submit" class="submit-btn">Send Message</button>
        </form>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { reactive } from "vue";

const formData = reactive({
  name: "",
  email: "",
  subject: "",
  message: "",
});

const handleSubmit = () => {
  // TODO: Implement form submission logic
  console.log("Form submitted:", formData);
};
</script>

<style scoped>
.contact-page {
  flex: 1;
  width: 100%;
  padding: 2rem 0;
}

.contact-content {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 1rem;
}

h1 {
  color: #4caf50;
  font-size: 2.5rem;
  margin-bottom: 2rem;
}

.section {
  margin-bottom: 2.5rem;
}

h2 {
  color: #81c784;
  font-size: 1.8rem;
  margin-bottom: 1rem;
}

p {
  color: #e0e0e0;
  line-height: 1.6;
  margin-bottom: 1rem;
}

.contact-details {
  background: rgba(255, 255, 255, 0.05);
  padding: 1.5rem;
  border-radius: 8px;
  margin-top: 1rem;
}

.contact-form {
  background: rgba(255, 255, 255, 0.05);
  padding: 2rem;
  border-radius: 8px;
}

.form-group {
  margin-bottom: 1.5rem;
}

label {
  display: block;
  color: #e0e0e0;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
}

input,
textarea {
  width: 100%;
  padding: 0.8rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.05);
  color: #e0e0e0;
  font-size: 1rem;
}

input:focus,
textarea:focus {
  outline: none;
  border-color: #4caf50;
  box-shadow: 0 0 0 2px rgba(76, 175, 80, 0.2);
}

.submit-btn {
  background: linear-gradient(135deg, #4caf50, #45a049);
  color: white;
  border: none;
  padding: 1rem 2rem;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
  width: 100%;
}

.submit-btn:hover {
  background: linear-gradient(135deg, #45a049, #3d8b40);
  transform: translateY(-1px);
}
</style>
