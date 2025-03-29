<template>
  <div class="pricing-page">
    <h1>Pricing Plans</h1>
    <div class="pricing-grid">
      <div v-for="(plan, index) in plans" :key="index" class="pricing-card">
        <h3>{{ plan.name }}</h3>
        <div class="price">${{ plan.price }}/mo</div>
        <ul>
          <li v-for="(feature, fIndex) in plan.features" :key="fIndex">
            {{ feature }}
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const plans = [
  {
    name: "Starter",
    price: "49",
    features: [
      "Basic multi-cloud support",
      "Up to 3 regions",
      "24/7 monitoring",
    ],
  },
  {
    name: "Professional",
    price: "149",
    features: [
      "Advanced multi-cloud support",
      "Unlimited regions",
      "Priority support",
    ],
  },
];
</script>

<style scoped>
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 2rem;
  margin-top: 2rem;
}

.pricing-card {
  padding: 2rem;
  background: #242424;
  border-radius: 8px;
  border: 1px solid #333;
  text-align: center;
}

.price {
  font-size: 2rem;
  margin: 1rem 0;
  color: var(--primary-color);
}
</style>
