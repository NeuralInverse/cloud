INSERT INTO notification_templates (id, name, title_template, body_template, "group", actions)
VALUES (
    'b22c0c32-9f73-4b96-8d5e-4f1a2c3d4e5f',
    'Workspace Stopped — Billing',
    E'Workspace "{{.Labels.name}}" stopped — billing action required',
    E'Hi {{.UserName}}\n\nYour workspace **{{.Labels.name}}** has been stopped.\n\n{{if eq .Labels.reason "free-trial-exhausted"}}Your free trial credit has been used up. Add a payment method to resume and get **$10 free compute credit** every month.{{else if eq .Labels.reason "unpaid-invoice"}}You have an outstanding invoice. Please settle your balance to resume your workspaces.{{else}}Your last payment failed. Please update your payment method to resume your workspaces.{{end}}',
    'Workspace Events',
    '[
        {
            "label": "Go to billing",
            "url": "{{.Labels.billing_url}}"
        }
    ]'::jsonb
);
