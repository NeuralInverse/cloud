INSERT INTO notification_templates (id, name, title_template, body_template, "group", actions)
VALUES (
    'b22c0c32-9f73-4b96-8d5e-4f1a2c3d4e5f',
    'Workspace Quota Exceeded',
    E'Workspace "{{.Labels.name}}" stopped — free trial exhausted',
    E'Hi {{.UserName}}\n\nYour workspace **{{.Labels.name}}** has been stopped because your free trial credit has been used up.\n\nAdd a payment method to resume your workspace and get **$10 free compute credit** every month.',
    'Workspace Events',
    '[
        {
            "label": "Add payment method",
            "url": "{{.Labels.billing_url}}"
        }
    ]'::jsonb
);
