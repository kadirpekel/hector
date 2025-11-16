-- Knowledge Base Schema
CREATE TABLE IF NOT EXISTS knowledge_articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    author VARCHAR(100),
    status VARCHAR(20) DEFAULT 'published',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_knowledge_articles_status ON knowledge_articles(status);
CREATE INDEX idx_knowledge_articles_updated ON knowledge_articles(updated_at);

-- Sample Knowledge Base Articles
INSERT INTO knowledge_articles (title, content, category, author, status) VALUES
('Enterprise Security Policy', 
 'All employees must use strong passwords with at least 12 characters, including uppercase, lowercase, numbers, and special characters. Passwords must be changed every 90 days. Multi-factor authentication is required for all systems accessing sensitive data. VPN access is mandatory when working remotely.',
 'Security',
 'IT Security Team',
 'published'),

('Code Review Guidelines',
 'All code changes must be reviewed by at least two senior developers before merging. Code reviews should focus on: security vulnerabilities, performance implications, test coverage, and adherence to coding standards. Critical changes require approval from the architecture team.',
 'Development',
 'Engineering Lead',
 'published'),

('Deployment Process',
 'Production deployments follow a strict process: 1) Code review and approval, 2) Automated testing in staging environment, 3) Security scan, 4) Change management ticket, 5) Deployment during maintenance window, 6) Post-deployment verification. Rollback plan must be documented before deployment.',
 'Operations',
 'DevOps Team',
 'published'),

('Data Retention Policy',
 'Customer data is retained for 7 years after account closure. Transaction logs are kept for 3 years. Application logs are retained for 90 days. All data deletion must be approved by the data protection officer and logged in the compliance system.',
 'Compliance',
 'Legal Team',
 'published'),

('Incident Response Procedure',
 'When a security incident is detected: 1) Immediately notify the security team via incident@company.com, 2) Isolate affected systems, 3) Document timeline and impact, 4) Begin forensic analysis, 5) Notify stakeholders within 1 hour, 6) Prepare incident report within 24 hours. Critical incidents require CISO notification within 15 minutes.',
 'Security',
 'Security Operations',
 'published'),

('API Rate Limiting Standards',
 'All public APIs must implement rate limiting: 100 requests per minute per IP for unauthenticated endpoints, 1000 requests per minute per authenticated user. Rate limit headers must be included in all responses. Exceeding limits returns HTTP 429 with Retry-After header.',
 'Development',
 'API Team',
 'published'),

('Database Backup Strategy',
 'Production databases are backed up every 6 hours with full backups daily. Backups are retained for 30 days on-site and 90 days off-site. Backup restoration is tested monthly. All backups are encrypted at rest. Backup integrity is verified after each backup operation.',
 'Operations',
 'Database Team',
 'published'),

('Customer Support Escalation',
 'Tier 1 support handles general inquiries. Tier 2 handles technical issues requiring deeper investigation. Tier 3 involves engineering team for bugs or feature requests. Critical issues affecting multiple customers are escalated immediately to the on-call engineer. SLA: Tier 1 response < 4 hours, Tier 2 < 2 hours, Tier 3 < 1 hour.',
 'Support',
 'Customer Success',
 'published'),

('Monitoring and Alerting',
 'All production services must have: CPU, memory, disk, and network monitoring. Application metrics for request rate, error rate, and latency. Alerts configured for: service downtime, error rate > 1%, latency p95 > 1s, disk usage > 80%. On-call rotation handles alerts 24/7.',
 'Operations',
 'SRE Team',
 'published'),

('Access Control Matrix',
 'Role-based access control: Developers have read access to staging, write access to their team repositories. Operations have full access to infrastructure. Security team has read-only access to all systems. External contractors have time-limited, scoped access. All access is logged and audited quarterly.',
 'Security',
 'Identity Management',
 'published');

-- Update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_knowledge_articles_updated_at 
    BEFORE UPDATE ON knowledge_articles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

