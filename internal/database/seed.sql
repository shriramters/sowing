-- Clear existing data to make the script idempotent
DELETE FROM revisions;
DELETE FROM pages;
DELETE FROM silos;
DELETE FROM users;
DELETE FROM identities;

-- Reset autoincrement counters
DELETE FROM sqlite_sequence WHERE name IN ('revisions', 'pages', 'silos', 'users', 'identities');

-- =============================================
--                Seed Data
-- =============================================

-- Default User
INSERT INTO users (id, username, display_name) VALUES (1, 'admin', 'Admin User');

-- Infrastructure Silo
INSERT INTO silos (id, slug, name, cover_image) VALUES (1, 'infrastructure', 'Infrastructure', 'https://picsum.photos/720/480');

-- Pages & Revisions for the Infrastructure Silo
-- The pages are created first, then the revisions, then the pages are updated with their revision IDs.
-- This is a bit verbose but avoids foreign key issues during seeding.

-- Page Definitions
INSERT INTO pages (id, silo_id, parent_id, slug, title, position, current_revision_id) VALUES
(1, 1, NULL, 'home',       'Home',              0, 1),
(2, 1, NULL, 'servers',    'Servers',           1, 2),
(3, 1, 2,    'web-server', 'Web Server',        0, 3),
(4, 1, 2,    'db-server',  'Database Server',   1, 4),
(5, 1, NULL, 'networking', 'Networking',        2, 5),
(6, 1, 5,    'dns',        'DNS Configuration', 0, 6);

-- Revision Content
INSERT INTO revisions (id, page_id, author_id, comment, content) VALUES
(1, 1, 1, 'Initial creation',
'* Welcome to the Infrastructure Silo!

This silo contains all documentation related to our production and staging infrastructure.

**Sections:**
 - [[./servers][Servers]]: Details on individual server setup and maintenance.
 - [[./networking][Networking]]: Information on our network topology, VLANs, and DNS.
'),

(2, 2, 1, 'Initial creation',
'* Servers Overview

This section documents all physical and virtual servers.

| Hostname      | IP Address      | Role            | OS         |
|---------------+-----------------+-----------------+------------|
| ~web-01~      | =192.168.1.10=  | Web Server      | Ubuntu 22.04 |
| ~db-01~       | =192.168.1.20=  | Database Server | Ubuntu 22.04 |
| ~utility-01~  | =192.168.1.30=  | Utility         | CentOS 9   |
'),

(3, 3, 1, 'Initial creation',
'* Web Server Setup (web-01)

Our primary web server runs Nginx.

**Nginx Configuration**
Here is the base configuration for our standard web applications.

#+BEGIN_SRC nginx
server {
    listen 80;
    server_name example.com;

    root /var/www/html;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
#+END_SRC
'),

(4, 4, 1, 'Initial creation',
'* Database Server Setup (db-01)

We use PostgreSQL for our primary database.

**Setup Checklist**
- [X] Install PostgreSQL 14
- [X] Configure ~pg_hba.conf~ for remote access
- [ ] Set up daily backups with ~pg_dump~
- [ ] Configure monitoring with Prometheus
'),

(5, 5, 1, 'Initial creation',
'* Networking Infrastructure

This page outlines the high-level network design.

*** VLANs
Our network is segmented into the following VLANs:
 - *VLAN 10:* Servers
 - *VLAN 20:* Desktops
 - *VLAN 30:* VoIP
 - *VLAN 99:* Management
'),

(6, 6, 1, 'Initial creation',
'* DNS Configuration

We use an internal DNS server for local name resolution.

*** Common Records
 - /A Records/ :: Map hostnames to IPv4 addresses.
 - /CNAME Records/ :: Create aliases for hostnames.
 - /MX Records/ :: Define mail servers.
');

