<template>
  <div class="contact-page">
    <h1>Contact Us</h1>
    <div class="contact-container">
      <form class="contact-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="name">Name</label>
          <input id="name" v-model="name" type="text" required />
        </div>
        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" required />
        </div>
        <div class="form-group">
          <label for="message">Message</label>
          <textarea id="message" v-model="message" required></textarea>
        </div>
        <button type="submit">Send Message</button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
const name = ref("");
const email = ref("");
const message = ref("");

const handleSubmit = () => {
  // TODO: Implement contact form submission
  console.log("Contact form submitted");
};
</script>

<style scoped>
.contact-container {
  max-width: 600px;
  margin: 0 auto;
}

.contact-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

input,
textarea {
  padding: 0.75rem;
  background: #242424;
  border: 1px solid #333;
  border-radius: 4px;
  color: #e0e0e0;
}

textarea {
  min-height: 150px;
}

button {
  padding: 0.75rem;
  background: var(--primary-color);
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
}
</style>
