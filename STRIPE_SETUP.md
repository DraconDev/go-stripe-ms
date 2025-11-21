# Stripe Setup Guide

## 🔑 Required Information from Stripe

### 1. API Keys (Both Test & Live)

**Location:** https://dashboard.stripe.com/test/apikeys

**Test Mode Keys** (Start here!)
```bash
STRIPE_SECRET_KEY=sk_test_51...      # Backend only - NEVER expose to frontend!
STRIPE_PUBLISHABLE_KEY=pk_test_51... # Frontend safe - used in browser/mobile
```

**Live Mode Keys** (Production only!)  
```bash
STRIPE_SECRET_KEY=sk_live_51...      # Production backend
STRIPE_PUBLISHABLE_KEY=pk_live_51... # Production frontend
```

⚠️ **CRITICAL:** Never commit secret keys to git! Always use environment variables.

---

### 2. Webhook Signing Secret

**Location:** https://dashboard.stripe.com/webhooks

**Setup Steps:**
1. Click "Add endpoint"
2. Enter endpoint URL: `https://yourdomain.com/webhooks/stripe`
3. Select these events (minimum):
   - ✅ `checkout.session.completed`
   - ✅ `customer.subscription.created`
   - ✅ `customer.subscription.updated`
   - ✅ `customer.subscription.deleted`
   - ✅ `invoice.payment_succeeded`
   - ✅ `invoice.payment_failed`
4. Copy webhook signing secret:

```bash
STRIPE_WEBHOOK_SECRET=whsec_...
```

**Local Testing with Stripe CLI:**
```bash
# Install
brew install stripe/stripe-cli/stripe

# Login
stripe login

# Forward webhooks to local server
stripe listen --forward-to localhost:8080/webhooks/stripe

# Copy the webhook secret shown (whsec_...)
```

---

### 3. Products & Prices

**Location:** https://dashboard.stripe.com/products

**For Each Product You Want to Sell:**

#### Subscription Example:
- Product Name: "Premium Plan"
- Product ID: `prod_ABC123` (auto-generated)
- Price (Monthly): `price_123ABC` ← **Use this in API calls**
- Price (Yearly): `price_456DEF` ← **Use this in API calls**

#### One-Time Purchase Example:
- Product Name: "E-book Course"  
- Product ID: `prod_XYZ789`
- Price: `price_789XYZ` ← **Use this in API calls**

**API Usage:**
```json
POST /api/v1/checkout/subscription
{
  "user_id": "user_123",
  "email": "user@example.com",
  "product_id": "prod_ABC123",
  "price_id": "price_123ABC",      ← From Stripe Dashboard
  "success_url": "https://myapp.com/success",
  "cancel_url": "https://myapp.com/cancel"
}
```

---

## 🧪 Stripe Test Cards

Use these for testing in **Test Mode**:

| Card Number | Scenario |
|-------------|----------|
| `4242 4242 4242 4242` | ✅ Success (any CVC, future date) |
| `4000 0000 0000 0002` | ❌ Card declined |
| `4000 0025 0000 3155` | 🔐 Requires 3D Secure authentication |
| `4000 0000 0000 9995` | ❌ Insufficient funds |

CVC: Any 3 digits (e.g., 123)  
Expiry: Any future date (e.g., 12/25)

---

## 📋 Complete Environment Variables

```bash
# Database
DATABASE_URL=postgresql://user:password@host:5432/database

# Stripe (Test Mode - Development)
STRIPE_SECRET_KEY=sk_test_51...
STRIPE_WEBHOOK_SECRET=whsec_...

# Stripe (Live Mode - Production Only!)
# STRIPE_SECRET_KEY=sk_live_51...
# STRIPE_WEBHOOK_SECRET=whsec_...

# Server
HTTP_PORT=8080
LOG_LEVEL=info

# Optional: CORS (only if browser calls API directly)
# CORS_ALLOWED_ORIGINS=https://myapp.com,https://app.myapp.com

# Optional: Environment
ENVIRONMENT=development
```

---

## 🎯 Quick Start Checklist

- [ ] Create Stripe account at https://stripe.com
- [ ] Get **Test API keys** from dashboard
- [ ] Create your first **product** and **price**
- [ ] Set up **webhook endpoint** (use Stripe CLI for local testing)
- [ ] Copy **webhook secret**
- [ ] Add all variables to `.env` file
- [ ] Test with test card `4242 4242 4242 4242`
- [ ] Verify webhook events are received

---

## ⚠️ Production Checklist

Before going live:

- [ ] Switch to **Live API keys**
- [ ] Update webhook endpoint to production URL
- [ ] Get new **Live webhook secret**
- [ ] Test in production with real card (small amount)
- [ ] Set up **Stripe tax** configuration (if needed)
- [ ] Configure **receipt emails** in Stripe settings
- [ ] Review **Stripe radar rules** (fraud prevention)
- [ ] Complete **Stripe verification** requirements

---

## 📚 Helpful Links

- **Stripe Dashboard:** https://dashboard.stripe.com
- **API Documentation:** https://stripe.com/docs/api
- **Webhook Events:** https://stripe.com/docs/api/events/types
- **Test Cards:** https://stripe.com/docs/testing
- **CLI Documentation:** https://stripe.com/docs/stripe-cli
