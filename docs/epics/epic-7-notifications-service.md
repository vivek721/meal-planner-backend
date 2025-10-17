# Epic 7: Notifications & Admin Dashboard
## Email Service & Administrative Features

---

**Epic ID:** EPIC-7
**Priority:** P2 (Medium)
**Estimated Effort:** 25 hours
**Sprint:** Week 7
**Owner:** Backend Developer
**Status:** Not Started
**Dependencies:** Epic 2 (Authentication), Epic 4 (Meal Planning)

---

## Overview

Implement email notification system for user engagement and build admin dashboard backend for content management and analytics.

## Goals

1. Set up email service (SendGrid or AWS SES)
2. Create email templates
3. Implement notification triggers
4. Build admin analytics endpoints
5. Add admin user management
6. Implement admin recipe moderation

## User Stories

### US-7.1: Send welcome email on registration

**Trigger:** User completes registration

**Template:** `welcome.html`

**Content:**
- Welcome message
- Getting started guide
- Link to browse recipes
- Link to create first meal plan

**Implementation:**
```typescript
async sendWelcomeEmail(user: User) {
  const template = await emailService.renderTemplate('welcome', {
    userName: user.name,
    dashboardUrl: 'https://app.mealplanner.com/dashboard',
  });

  await emailService.send({
    to: user.email,
    subject: 'Welcome to Meal Planner!',
    html: template,
  });
}
```

---

### US-7.2: Send weekly meal plan reminder

**Trigger:** Sunday 9 AM (user's timezone)

**Template:** `meal-plan-reminder.html`

**Content:**
- Reminder to plan upcoming week
- Button: "Plan This Week"
- AI-suggested recipes preview

**Scheduling:**
```typescript
// Cron job: Every Sunday at 9 AM
cron.schedule('0 9 * * 0', async () => {
  const users = await db.users.findActive();

  for (const user of users) {
    const hasPlannedThisWeek = await db.mealPlans.exists({
      userId: user.id,
      weekStart: getNextSunday(),
    });

    if (!hasPlannedThisWeek && user.preferences.notificationPreferences?.email?.weeklyReminder) {
      await emailService.sendWeeklyReminder(user);
    }
  }
});
```

---

### US-7.3: Send shopping list ready notification

**Trigger:** Shopping list generated

**Template:** `shopping-list-ready.html`

**Content:**
- Shopping list summary (item count)
- Link to view/share list
- Option to email list

---

### US-7.4: Send password reset email

**Trigger:** User requests password reset

**Template:** `password-reset.html`

**Content:**
- Reset link with token (1-hour expiry)
- Security warning (didn't request? ignore this email)

---

### US-7.5: Admin analytics dashboard

**Endpoint:** GET `/api/v1/admin/analytics`

**Response:**
```json
{
  "data": {
    "users": {
      "total": 10234,
      "active": 4567,
      "newThisWeek": 156
    },
    "recipes": {
      "total": 1234,
      "published": 1156
    },
    "mealPlans": {
      "totalCreated": 45678,
      "activeThisWeek": 3456
    },
    "engagement": {
      "avgMealsPerWeek": 12.5,
      "avgRecipesPerPlan": 8.3,
      "favoriteRate": 0.23
    },
    "topRecipes": [
      {
        "id": "recipe-001",
        "name": "Chicken Tacos",
        "timesUsed": 2345,
        "rating": 4.5
      }
    ]
  }
}
```

---

### US-7.6: Admin user management

**Endpoint:** GET `/api/v1/admin/users?page=1&limit=50&search=email`

**Response:**
```json
{
  "data": [
    {
      "id": "user-001",
      "email": "sarah@example.com",
      "name": "Sarah Johnson",
      "role": "user",
      "createdAt": "2025-09-15T10:00:00Z",
      "lastLoginAt": "2025-10-14T09:30:00Z",
      "mealPlansCount": 8,
      "favoritesCount": 23
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 10234
  }
}
```

---

## Technical Requirements

### Email Service Setup

**SendGrid Configuration:**
```typescript
import sgMail from '@sendgrid/mail';

sgMail.setApiKey(process.env.SENDGRID_API_KEY);

class EmailService {
  async send({ to, subject, html }) {
    await sgMail.send({
      to,
      from: 'noreply@mealplanner.com',
      subject,
      html,
      trackingSettings: {
        clickTracking: { enable: true },
        openTracking: { enable: true },
      },
    });
  }

  async renderTemplate(templateName, data) {
    const template = await fs.readFile(`templates/${templateName}.html`, 'utf-8');
    return Handlebars.compile(template)(data);
  }
}
```

**Email Queue (Bull):**
```typescript
import Queue from 'bull';

const emailQueue = new Queue('emails', {
  redis: {
    host: process.env.REDIS_HOST,
    port: 6379,
  },
});

emailQueue.process(async (job) => {
  await emailService.send(job.data);
});

// Usage
await emailQueue.add({
  to: 'user@example.com',
  subject: 'Welcome!',
  html: '<h1>Welcome</h1>',
});
```

---

## Email Templates

### welcome.html
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Welcome to Meal Planner</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
  <div style="background: #2563eb; color: white; padding: 20px; text-align: center;">
    <h1>Welcome to Meal Planner!</h1>
  </div>

  <div style="padding: 20px;">
    <p>Hi {{userName}},</p>

    <p>We're excited to have you on board! Meal Planner helps you plan delicious, nutritious meals and generate smart shopping lists.</p>

    <h3>Get Started:</h3>
    <ul>
      <li>Browse our recipe collection</li>
      <li>Create your first weekly meal plan</li>
      <li>Generate a shopping list</li>
      <li>Get AI-powered meal suggestions</li>
    </ul>

    <a href="{{dashboardUrl}}" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin-top: 20px;">
      Go to Dashboard
    </a>
  </div>

  <div style="background: #f3f4f6; padding: 20px; text-align: center; font-size: 12px;">
    <p>&copy; 2025 Meal Planner. All rights reserved.</p>
    <a href="{{unsubscribeUrl}}" style="color: #6b7280;">Unsubscribe</a>
  </div>
</body>
</html>
```

---

## API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/admin/users` | Admin | List all users |
| GET | `/admin/analytics` | Admin | Dashboard metrics |
| GET | `/admin/health` | Admin | System health |
| PATCH | `/admin/users/:id/role` | Admin | Update user role |
| DELETE | `/admin/users/:id` | Admin | Delete user account |

---

## Testing Requirements

```typescript
describe('Email Service', () => {
  it('should send welcome email', async () => {
    const mockSend = jest.spyOn(emailService, 'send');

    await authService.register({
      email: 'newuser@example.com',
      password: 'Password123!',
      name: 'New User',
    });

    expect(mockSend).toHaveBeenCalledWith(
      expect.objectContaining({
        to: 'newuser@example.com',
        subject: 'Welcome to Meal Planner!',
      })
    );
  });

  it('should handle email failures gracefully', async () => {
    jest.spyOn(emailService, 'send').mockRejectedValue(new Error('SendGrid error'));

    // Registration should still succeed even if email fails
    const user = await authService.register({
      email: 'test@example.com',
      password: 'Password123!',
      name: 'Test User',
    });

    expect(user).toBeDefined();
  });
});

describe('Admin Analytics', () => {
  it('should return dashboard metrics', async () => {
    const res = await request(app)
      .get('/api/v1/admin/analytics')
      .set('Authorization', `Bearer ${adminToken}`);

    expect(res.status).toBe(200);
    expect(res.body.data.users.total).toBeGreaterThan(0);
  });

  it('should require admin role', async () => {
    const res = await request(app)
      .get('/api/v1/admin/analytics')
      .set('Authorization', `Bearer ${userToken}`);

    expect(res.status).toBe(403);
  });
});
```

---

## Acceptance Criteria

- [ ] Email service configured (SendGrid or SES)
- [ ] 5 email templates created and tested
- [ ] Welcome email sent on registration
- [ ] Weekly reminder cron job scheduled
- [ ] Password reset email working
- [ ] Admin analytics endpoint implemented
- [ ] Admin user management functional
- [ ] Email queue handles failures gracefully
- [ ] Unsubscribe links in all emails

---

## Timeline

| Day | Tasks | Hours |
|-----|-------|-------|
| **Mon** | Email service setup, templates | 8 |
| **Tue** | Welcome, reminder, reset emails | 8 |
| **Wed** | Admin analytics endpoint | 5 |
| **Thu** | Admin user management | 4 |

**Total:** 25 hours

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
