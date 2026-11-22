# Event & Notification Connectivity
Event → notifikasi/audit terverifikasi:
- login (audit), login.locked, new-IP login notify, MFA enable/verify.
- client.create/update/merge/specimen; identity status/history.
- request.create/move + assignee notify (async pool), SLA escalation via remind.RunOnce.
- deed.review/approve/sign/version + repertorium.insert audit (020).
- doc upload/preview/download/share/verify/field/bulk + doc_access_logs.
- invoice.create/payment.create/bank.import/bank.match/bank.manual/void.
- task.recur, will.register/review/open/handover, mandate.create, backup.now.
- digest pagi + ultah + ID expiry + will review via remind.RunPeople/Digest.
Anti-spam: dedupe 24 jam per (user,title,body). Retry aman: best-effort Sender tidak menggagalkan request.
