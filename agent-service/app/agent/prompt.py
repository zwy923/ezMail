CLASSIFICATION_PROMPT = """You are an email classification assistant. 
Analyze the following email and classify it into one of these categories:
- finance: emails related to payments, invoices, bills, banking
- schedule: emails related to meetings, appointments, calendar events
- other: everything else

Email Subject: {subject}
Email Body: {body}

Respond with ONLY the category name (finance, schedule, or other).
"""

