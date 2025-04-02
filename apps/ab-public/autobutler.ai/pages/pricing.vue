<template>
  <PageContainer>
    <h1>Pricing Plans</h1>
    <div class="pricing-grid">
      <div v-for="plan in plans" :key="plan.name" class="pricing-card">
        <h2>{{ plan.name }}</h2>
        <div class="price">
          <span class="amount">${{ plan.price }}</span>
          <span class="period">/month</span>
        </div>
        <ul class="features">
          <li v-for="feature in plan.features" :key="feature">{{ feature }}</li>
        </ul>
        <button class="select-plan">Select Plan</button>
      </div>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
const plans = [
  {
    name: 'Starter',
    price: 29,
    features: [
      'Basic Automation',
      '5 Workflows',
      'Email Support',
      'Basic Analytics'
    ]
  },
  {
    name: 'Professional',
    price: 99,
    features: [
      'Advanced Automation',
      'Unlimited Workflows',
      'Priority Support',
      'Advanced Analytics'
    ]
  },
  {
    name: 'Enterprise',
    price: 299,
    features: [
      'Custom Solutions',
      'Dedicated Support',
      'SLA Guarantee',
      'Custom Analytics'
    ]
  }
];
</script>

<style scoped>
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  margin-top: 2rem;
}

.pricing-card {
  padding: 2rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  text-align: center;
  transition: all 0.3s ease;
}

.pricing-card:hover {
  border-color: rgba(0, 255, 170, 0.3);
  transform: translateY(-4px);
}

.price {
  margin: 1.5rem 0;
}

.amount {
  font-size: 3rem;
  font-weight: bold;
}

.period {
  color: rgba(255, 255, 255, 0.7);
}

.features {
  list-style: none;
  padding: 0;
  margin: 2rem 0;
}

.features li {
  margin: 0.5rem 0;
  color: rgba(255, 255, 255, 0.8);
}

.select-plan {
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.2),
    rgba(0, 187, 255, 0.2)
  );
  border: 1px solid rgba(0, 255, 170, 0.3);
  padding: 0.8rem 2rem;
  border-radius: 4px;
  color: #fff;
  cursor: pointer;
  transition: all 0.3s ease;
}

.select-plan:hover {
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.3),
    rgba(0, 187, 255, 0.3)
  );
}
</style> 