# Crane Thorough-Examination Reference Requirements 
 
Status: Nonbinding reference derived from a user-provided example. Do not import its identities, certificate data, signatures, branding, contacts, or claimed regulatory basis. 
 
## Reusable Report Structure 
1. Certificate control: report identifier, page numbering, issue/examination dates, regulation or procedure version, and future verification reference. 
2. Equipment identity: equipment class, manufacturer/model, serial numbers, year, capacity, hook/block data, dimensions, and assignment/site context. 
3. Examination parameters: load-test geometry, radius, boom length, safe working load, proof load, instruments, and method. 
4. Condition checklist: sectioned component checks with Pass, Fail, Not Applicable, comment, defect, corrective action, and follow-up fields. 
5. Evidence: general equipment views, identifier plate, critical component closeups, charts/indicators, measurements, and load-test or NDT evidence. 
6. Decision: defect status, repair requirement, safe-to-operate recommendation, inspector and verifier attestations, issue/expiry, and server-issued certificate result. 
 
## Suggested INTEGIN Work-Package Sections 
- Equipment profile and assignment context. 
- Operator area, boom, load indicator, chassis, wheels, outriggers, slewing, counterweights, hook/block, winches, wire rope, and safety devices. 
- Load test, NDT, or measurement sub-report linked by stable equipment and inspection identifiers. 
- Defect/corrective-action workflow and evidence metadata. 
 
## INTEGIN Design Rules 
Use stable field identifiers and section versions; render native Pass/Fail/NA controls offline; preserve comments and evidence metadata; and bind every submission to the immutable work-package version. 
The server, not the field app, decides authoritative acceptance, certificate issuance, QR verification, and any safe-to-operate or compliance status. 
 
## Source-Verified Field Inventory 
The example is a reusable field taxonomy, not a form to copy. The work package should model identity and context as typed attributes, technical tests as repeatable unit-aware rows, and component observations as individual control records. 
 
1. Equipment and context fields: class, manufacturer/model, identifiers, year, capacities, hook and wire-rope data, boom range, inspection site, assignment, and inspection trigger. 
2. Test-matrix fields: test point, configuration, boom length, radius, SWL, proof load, result, unit, method, and instrument reference. 
3. Component-control fields: stable section ID, stable item ID, Pass/Fail/Not Applicable status, comment, and linked defect. A blank result is not valid for a required item. 
4. Defect fields: severity, description, immediate-danger flag, correction requirement, due-by date, and resolution evidence. A failed control must create a separately auditable finding. 
5. Attestation fields: inspector identity, qualification basis, signature event, reviewer/authenticator, and next-due proposal. 
 
## Evidence and Issuance Boundaries 
Request evidence by purpose: overall equipment, identifier plate, critical component close-up, configuration or indicator, test configuration, and measurement or NDT result. Store capture time, device-key reference, field link, content hash, capture mode, and optional measurement metadata. 
The future certificate renderer compiles a server-issued immutable view from control metadata, inspection rows, findings, evidence links, test annexes, attestations, and the authority decision. Any QR verification reference must disclose only an intentionally published record. 
The example does not determine INTEGIN legal compliance, safety thresholds, qualification rules, due intervals, or safe-to-operate outcomes. Those remain authority-reviewed, versioned package policies. 
