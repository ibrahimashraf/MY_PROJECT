/* ==========================================================================
   INTEGIN Interactive Onboarding Wizard Controller & State Machine
   Handles:
   - 5-step stepper navigation
   - Smart Resilient Certificate Template Parser (Supports any custom tokens)
   - Dual-layer auto-save & crash/power-cut recovery simulation
   - QR code device-pairing simulation (Ed25519 key generation)
   - Realistic Interactive Engineering Physics Sandbox (Load & Defect Simulator)
   - Dedicated Optical Vector Document Zoom Engine (True Resolution Zoom)
   - Instant English / Arabic (RTL) localization toggle
   ========================================================================== */

(function () {
  'use strict';

  // --- Localization Dictionary (EN / AR) ---
  const i18n = {
    en: {
      tagline: "Integrated Inspection & Assurance",
      autoSaved: "Draft Auto-Persisted",
      saving: "Persisting Draft...",
      simulateCrash: "Simulate Crash / Restart",
      trustReadiness: "Trust & Regulatory Readiness",
      step1: "Legal Identity",
      step2: "Accreditation",
      step3: "Certificate Rules",
      step4: "Disciplines",
      step5: "Edge Pairing & Launch",
      step1Title: "Organization Legal Identity",
      step1Desc: "Establish your company’s legal entity and tax identifiers to anchor Row-Level Security (RLS) data partitioning.",
      companyName: "Inspection Company Legal Name *",
      crNumber: "Commercial Registration (CR) Number *",
      vatNumber: "VAT / Tax Identification (ZATCA) *",
      country: "Jurisdiction / Country *",
      subdomain: "Tenant Domain Slug",
      step2Title: "Accreditations, Badges & Official Stamp",
      step2Desc: "Configure your official accreditation marks (LEEA, ISO 17020, ASME) to be cryptographically embedded in issued certificates.",
      companyLogo: "Company Official Logo",
      dropLogoText: "Drag logo or click to upload PNG",
      officialStamp: "Digital Seal / Watermark Stamp",
      dropStampText: "Upload Official Inspection Stamp",
      selectAccreditations: "Select Recognized Accreditation Bodies",
      step3Title: "Certificate Governance & Numbering",
      step3Desc: "Define strict legal certificate formats and enforce whether single or four-eyes sign-off is required.",
      certPrefixPattern: "Certificate Numbering Template Pattern",
      insertTag: "Insert Dynamic Tags:",
      livePreview: "Live Rendered Sample:",
      signingPolicy: "Technical Signing Authority Policy",
      fourEyesTitle: "Mandatory Four-Eyes Verification (Recommended)",
      fourEyesDesc: "Inspector completes field inspection on tablet ➔ Technical Authority/QA Manager must review and counter-sign before official PDF release.",
      directSignTitle: "Direct Field Release",
      directSignDesc: "Authorized Level III inspectors can issue legally binding certificates directly upon sealing tablet report.",
      step4Title: "Active Inspection Disciplines",
      step4Desc: "Enable standardized test packages with automated engineering calculation limits.",
      cranesTitle: "Mobile & Crawler Cranes",
      cranesDesc: "Boom deflection, load moment indicators (LMI), outrigger hydraulics, wire rope defect scoring (ISO 4309).",
      tackleTitle: "Lifting Accessories & Shackles",
      tackleDesc: "Webbing slings, chain blocks, bow shackles, proof load tests, color coding schemes.",
      ndtTitle: "Non-Destructive Testing (NDT)",
      ndtDesc: "Magnetic Particle (MPI), Dye Penetrant (DPT), and Ultrasonic Thickness Gauging (UT).",
      vesselsTitle: "Pressure Vessels & Boilers",
      vesselsDesc: "Hydrostatic pressure testing, relief valve calibration, internal corrosion logging.",
      step5Title: "Live Inspection Simulation & Dynamic Physics Sandbox",
      step5Desc: "Control real-time engineering parameters, trigger overload/fault alerts, pair edge tablets, and see the official certificate generate live.",
      qrPairingTitle: "1. Field Tablet Scan-to-Pair",
      qrPairingDesc: "Scan with the INTEGIN Field App to generate local hardware keys in the device Secure Keystore.",
      awaitingScan: "Awaiting Tablet Scan...",
      pairedStatus: "✅ Tablet Paired: Tariq-Tab4-Active (Ed25519 Bound)",
      simulateScanBtn: "Simulate Tablet QR Scan",
      sandboxTitle: "2. Live Crane Inspection Sandbox",
      sandboxDesc: "Simulate load tests and defect evaluations in real-time. Altering parameters dynamically impacts the certificate status.",
      issueCertBtn: "Cryptographically Issue Live Certificate",
      prevBtn: "← Previous",
      nextBtn: "Continue Next →",
      resetBtn: "Reset Setup",
      finishBtn: "Complete Onboarding & Enter Dashboard"
    },
    ar: {
      tagline: "المنظومة المتكاملة للفحص والتأكيد الهندسي",
      autoSaved: "تم الحفظ التلقائي للمسودة",
      saving: "جاري الحفظ المشفر...",
      simulateCrash: "محاكاة انقطاع الطاقة / الإغلاق",
      trustReadiness: "جاهزية الامتثال والثقة النظامية",
      step1: "الهوية القانونية",
      step2: "الاعتمادات والشعارات",
      step3: "حوكمة الشهادات",
      step4: "مجالات الفحص",
      step5: "ربط الأجهزة والإطلاق",
      step1Title: "الهوية القانونية لجهة الفحص",
      step1Desc: "تسجيل الكيان القانوني وبيانات السجل التجاري والضريبة لترسيخ عزل البيانات على مستوى الصفوف (RLS).",
      companyName: "الاسم القانوني لشركة الفحص *",
      crNumber: "رقم السجل التجاري *",
      vatNumber: "الرقم الضريبي (هيئة الزكاة والضريبة) *",
      country: "الدولة / النطاق القضائي *",
      subdomain: "رمز النطاق الخاص بالمؤسسة",
      step2Title: "الاعتمادات المهنية والأختام الرسمية",
      step2Desc: "إعداد شارات الاعتماد (LEEA، ISO 17020، ASME) لتضمينها برمجياً في الشهادات المعتمدة.",
      companyLogo: "شعار الشركة الرسمي",
      dropLogoText: "اسحب الشعار أو انقر للتحميل (PNG)",
      officialStamp: "الختم الرقمي / العلامة المائية",
      dropStampText: "تحميل الختم الرسمي للفحص",
      selectAccreditations: "تحديد جهات الاعتماد المعترف بها",
      step3Title: "حوكمة وترقيم الشهادات الهندسية",
      step3Desc: "تحديد صيغة الأرقام التسلسلية الرسمية وسياسة التوقيع الرباعي (Four-Eyes Principle).",
      certPrefixPattern: "نمط ترقيم شهادات الفحص",
      insertTag: "إدراج وسوم ديناميكية:",
      livePreview: "معاينة حية مفسرة:",
      signingPolicy: "سياسة اعتماد التوقيع الفني",
      fourEyesTitle: "التحقق الثنائي الإلزامي (موصى به)",
      fourEyesDesc: "يوقع المفتش في الميدان على الجهاز اللوحي ➔ يراجع المدير الفني ويعتمد قبل إصدار النسخة النهائية.",
      directSignTitle: "الإصدار الميداني المباشر",
      directSignDesc: "يسمح للمفتشين المعتمدين من المستوى الثالث بإصدار الشهادات فور إغلاق التقرير بالميدان.",
      step4Title: "مجالات وتخصصات الفحص النشطة",
      step4Desc: "تفعيل حزم الفحص القياسية المزودة بحدود الحسابات الهندسية التلقائية.",
      cranesTitle: "الرافعات المتحركة والمجنزرة",
      cranesDesc: "انحراف الذراع، مؤشرات عزم الحمل (LMI)، هيدروليك الركائز، وتقييم حبال السلك (ISO 4309).",
      tackleTitle: "أدوات وملحقات الرفع والشواكل",
      tackleDesc: "أربطة الرفع القماشية، الكتل السلسلية، الشواكل، واختبارات أحمال الإثبات وترميز الألوان.",
      ndtTitle: "الاختبارات غير الإتلافية (NDT)",
      ndtDesc: "الفحص بالجزيئات المغناطيسية (MPI)، السوائل النافذة (DPT)، وسماكة الموجات فوق الصوتية (UT).",
      vesselsTitle: "أوعية الضغط والغلايات",
      vesselsDesc: "اختبارات الضغط الهيدروستاتيكي، معايرة صمامات الأمان، وسجلات التآكل الداخلي.",
      step5Title: "محاكاة فحص حية مع التحكم الهندسي الديناميكي",
      step5Desc: "تحكم في معايير الفحص الواقعية، اختبر أحمال الإثبات وتآكل الحبال وشاهد شهادة الفحص تصدر فورياً.",
      qrPairingTitle: "١. مسح رمز الاستجابة السريعة للربط",
      qrPairingDesc: "امسح الرمز بواسطة تطبيق المفتش لتوليد المفاتيح في الشريحة الأمنية للجهاز (Secure Keystore).",
      awaitingScan: "بانتظار مسح الجهاز اللوحي...",
      pairedStatus: "✅ تم ربط الجهاز بنجاح: Tariq-Tab4-Active (Ed25519)",
      simulateScanBtn: "محاكاة مسح الجهاز اللوحي للرمز",
      sandboxTitle: "٢. محاكاة فحص رافعة حية (اختبار الحمل)",
      sandboxDesc: "تغيير معايير الحمل والعيوب ينعكس فورياً على نتيجة الشهادة والحالة الأمنية للمعدة.",
      issueCertBtn: "إصدار وتوقيع الشهادة الحية تشفيرياً",
      prevBtn: "← السابق",
      nextBtn: "المتابعة للتالي →",
      resetBtn: "إعادة ضبط الإعداد",
      finishBtn: "إنهاء الإعداد والدخول للوحة التحكم"
    }
  };

  // --- State Machine ---
  let currentLocale = 'en';
  let currentStep = 1;
  const totalSteps = 5;
  let devicePaired = false;
  let currentZoom = 1.0;

  // Realistic Physics Simulation State
  let appliedLoad = 105; // Tonnes
  let wireRopeStatus = "MINOR"; // EXCELLENT, MINOR, BROKEN_WIRES, CORE_COLLAPSE

  const DEFAULT_PATTERN = "APEX-{DISCIPLINE}-{YEAR}-{SEQ:5}";

  const state = {
    companyName: "Apex Industrial Inspection Services LLC",
    crNumber: "1010894218",
    vatNumber: "300192837400003",
    country: "SA",
    subdomain: "apex",
    certPattern: DEFAULT_PATTERN,
    signPolicy: "FOUR_EYES",
    accreditations: ["LEEA", "ISO_17020"],
    disciplines: ["MOBILE_CRANES", "LIFTING_TACKLE"],
    currentStep: 1,
    devicePaired: false
  };

  const STORAGE_KEY = "integin_onboarding_draft_v1";

  // --- Initialization ---
  function init() {
    loadPersistedState();
    bindEvents();
    renderStep(state.currentStep);
    updateCertificatePreview();
    updateTrustScore();
    syncUIElements();
    updatePhysicsVerdict();
  }

  // --- Persistence & Crash Recovery ---
  function saveState() {
    state.currentStep = currentStep;
    state.companyName = document.getElementById('companyName')?.value || state.companyName;
    state.crNumber = document.getElementById('crNumber')?.value || state.crNumber;
    state.vatNumber = document.getElementById('vatNumber')?.value || state.vatNumber;
    state.country = document.getElementById('countrySelect')?.value || state.country;
    state.subdomain = document.getElementById('subdomainSlug')?.value || state.subdomain;
    const certPatternInput = document.getElementById('certPattern');
    state.certPattern = certPatternInput?.value || state.certPattern;
    state.devicePaired = devicePaired;

    // Save Signing Policy
    const selectedRadio = document.querySelector('input[name="signPolicy"]:checked');
    if (selectedRadio) {
      state.signPolicy = selectedRadio.value;
    }

    // Save Disciplines
    const checkedDisciplines = [];
    document.querySelectorAll('input[name="discipline"]:checked').forEach(cb => {
      checkedDisciplines.push(cb.value);
    });
    state.disciplines = checkedDisciplines;

    // Save Accreditations
    const checkedBadges = [];
    document.querySelectorAll('input[name="accreditation"]:checked').forEach(cb => {
      checkedBadges.push(cb.value);
    });
    state.accreditations = checkedBadges;

    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
      triggerSaveIndicator();
    } catch (e) {
      console.warn("Local storage write failed", e);
    }
  }

  function loadPersistedState() {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      try {
        const parsed = JSON.parse(raw);
        Object.assign(state, parsed);
        currentStep = state.currentStep || 1;
        devicePaired = state.devicePaired || false;

        const cName = document.getElementById('companyName');
        const crNum = document.getElementById('crNumber');
        const vatNum = document.getElementById('vatNumber');
        const cSelect = document.getElementById('countrySelect');
        const subSlug = document.getElementById('subdomainSlug');
        const certPatternInput = document.getElementById('certPattern');

        if (cName) cName.value = state.companyName || "";
        if (crNum) crNum.value = state.crNumber || "";
        if (vatNum) vatNum.value = state.vatNumber || "";
        if (cSelect) cSelect.value = state.country || "SA";
        if (subSlug) subSlug.value = state.subdomain || "apex";
        if (certPatternInput) certPatternInput.value = state.certPattern || DEFAULT_PATTERN;

        if (devicePaired) {
          markDevicePairedUI();
        }
      } catch (e) {
        console.error("State hydration error", e);
      }
    }
  }

  function syncUIElements() {
    document.querySelectorAll('.policy-card').forEach(card => {
      const radio = card.querySelector('input[name="signPolicy"]');
      if (radio) {
        if (radio.value === state.signPolicy) {
          radio.checked = true;
          card.classList.add('selected');
        } else {
          radio.checked = false;
          card.classList.remove('selected');
        }
      }
    });

    document.querySelectorAll('.discipline-card').forEach(card => {
      const cb = card.querySelector('input[name="discipline"]');
      if (cb) {
        if (state.disciplines.includes(cb.value)) {
          cb.checked = true;
          card.classList.add('active');
        } else {
          cb.checked = false;
          card.classList.remove('active');
        }
      }
    });

    document.querySelectorAll('.badge-checkbox-card').forEach(card => {
      const cb = card.querySelector('input[name="accreditation"]');
      if (cb) {
        cb.checked = state.accreditations.includes(cb.value);
      }
    });
  }

  function triggerSaveIndicator() {
    const saveStatusText = document.getElementById('saveStatusText');
    if (!saveStatusText) return;
    saveStatusText.textContent = i18n[currentLocale].saving;
    setTimeout(() => {
      saveStatusText.textContent = i18n[currentLocale].autoSaved;
    }, 300);
  }

  // --- Step Navigation & Stepper UI ---
  function renderStep(step) {
    currentStep = step;
    
    for (let i = 1; i <= totalSteps; i++) {
      const panel = document.getElementById(`stepContent${i}`);
      const btn = document.getElementById(`stepBtn${i}`);
      if (panel) panel.classList.toggle('active', i === step);
      if (btn) {
        btn.classList.toggle('active', i === step);
        btn.classList.toggle('completed', i < step);
      }
    }

    const prevStepBtn = document.getElementById('prevStepBtn');
    const nextStepBtn = document.getElementById('nextStepBtn');
    if (prevStepBtn) prevStepBtn.disabled = (step === 1);
    if (nextStepBtn) {
      if (step === totalSteps) {
        nextStepBtn.textContent = i18n[currentLocale].finishBtn;
      } else {
        nextStepBtn.textContent = i18n[currentLocale].nextBtn;
      }
    }

    updateTrustScore();
    saveState();
  }

  function updateTrustScore() {
    const trustScore = document.getElementById('trustScore');
    const trustProgressBar = document.getElementById('trustProgressBar');

    let score = 20;
    if (currentStep >= 2) score += 20;
    if (currentStep >= 3) score += 20;
    if (currentStep >= 4) score += 20;
    if (currentStep >= 5 && devicePaired) score = 100;
    else if (currentStep >= 5) score = 80;

    if (trustScore) trustScore.textContent = `${score}%`;
    if (trustProgressBar) trustProgressBar.style.width = `${score}%`;
  }

  // --- Resilient Smart Pattern Evaluation Engine ---
  function parsePattern(pattern, disciplineCode, sampleSeq) {
    if (!pattern || pattern.trim() === '') return "APEX-CRN-2026-00001";

    const dCode = disciplineCode || "CRN";
    const seqNum = sampleSeq || 1;
    const now = new Date();
    const currentYear = now.getFullYear().toString();
    const currentMonth = String(now.getMonth() + 1).padStart(2, '0');
    const companyPrefix = (document.getElementById('subdomainSlug')?.value || "APEX").toUpperCase();

    let output = pattern;

    // 1. Replace Standard Tokens Case-Insensitively
    output = output.replace(/\{PREFIX\}/gi, companyPrefix);
    output = output.replace(/\{DISCIPLINE\}/gi, dCode);
    output = output.replace(/\{YEAR\}/gi, currentYear);
    output = output.replace(/\{MONTH\}/gi, currentMonth);

    // 2. Dynamically Match Sequence Tags: {SEQ}, {SEQ:4}, {SEQ:5}, {SEQ:6}, {SEQ:8}
    output = output.replace(/\{SEQ(?::(\d+))?\}/gi, (match, padLength) => {
      const pad = padLength ? parseInt(padLength, 10) : 5;
      return String(seqNum).padStart(pad, '0');
    });

    return output;
  }

  function updateCertificatePreview() {
    const certPatternInput = document.getElementById('certPattern');
    const certPreviewOutput = document.getElementById('certPreviewOutput');
    if (!certPatternInput || !certPreviewOutput) return;

    const rawPattern = certPatternInput.value;
    const parsedSample = parsePattern(rawPattern, "CRN", 1);
    certPreviewOutput.textContent = parsedSample;
  }

  // --- Dynamic Realistic Physics Verdict Calculator ---
  function updatePhysicsVerdict() {
    const simVerdictBox = document.getElementById('simVerdictBox');
    const verdictIcon = document.getElementById('verdictIcon');
    const verdictHeading = document.getElementById('verdictHeading');
    const verdictDetail = document.getElementById('verdictDetail');
    const appliedLoadVal = document.getElementById('appliedLoadVal');
    const wireRopeVal = document.getElementById('wireRopeVal');

    const isRTL = (currentLocale === 'ar');

    if (appliedLoadVal) {
      const pct = Math.round((appliedLoad / 100) * 100);
      appliedLoadVal.textContent = `${appliedLoad}.0 Tonnes (${pct}% ${appliedLoad > 110 ? 'OVERLOAD' : 'Proof'})`;
      appliedLoadVal.className = `control-val ${appliedLoad > 110 ? 'text-danger' : (appliedLoad >= 100 ? 'text-primary' : '')}`;
    }

    const isOverload = (appliedLoad > 115);
    const isRopeFailed = (wireRopeStatus === "BROKEN_WIRES" || wireRopeStatus === "CORE_COLLAPSE");
    const isFailed = (isOverload || isRopeFailed);

    if (simVerdictBox && verdictIcon && verdictHeading && verdictDetail) {
      if (isFailed) {
        simVerdictBox.className = "sim-live-verdict state-fail";
        verdictIcon.textContent = "✕";
        
        if (isOverload) {
          verdictHeading.textContent = isRTL 
            ? "⚠️ نتيجة فنية: رسوب - خطر الحمل الزائد (OVERLOAD HAZARD)" 
            : "⚠️ TECHNICAL VERDICT: FAILED - STRUCTURAL OVERLOAD HAZARD";
          verdictDetail.textContent = isRTL
            ? `الحمل المطبق (${appliedLoad} طن) تجاوز الحد الأقصى لاختبار الإثبات (110%). تم تفعيل قفل الأمان الهيدروليكي.`
            : `Applied load (${appliedLoad}T) exceeds maximum allowable 110% proof threshold. Crane structural safety margin compromised.`;
        } else {
          verdictHeading.textContent = isRTL 
            ? "⚠️ نتيجة فنية: رسوب - حبل السلك معيب (QUARANTINED)" 
            : "⚠️ TECHNICAL VERDICT: FAILED - ISO 4309 DISCARD CRITERIA MET";
          verdictDetail.textContent = isRTL
            ? "حبل سلك الرافعة تجاوز حدود الأسلاك المقطوعة / انهيار القلب الداخلي وفق ISO 4309. يجب استبدال الحبل فوراً وإيقاف المعدة."
            : "Wire rope has met mandatory discard criteria under ISO 4309 (Severe broken wires or core collapse). Asset must be placed in safety quarantine.";
        }
      } else {
        simVerdictBox.className = "sim-live-verdict";
        verdictIcon.textContent = "✓";
        verdictHeading.textContent = isRTL 
          ? "النتيجة الفنية: صالح للاستخدام ومطابق للمواصفات (PASSED)" 
          : "TECHNICAL VERDICT: SAFE & FIT FOR OPERATION (PASSED)";
        verdictDetail.textContent = isRTL
          ? `تم اختبار الحمل بنجاح عند (${appliedLoad} طن). سلامة الركائز والذراع التلسكوبي ومؤشر LMI موثقة بالكامل.`
          : `Proof load test of ${appliedLoad}T verified with zero permanent deflection. Outriggers, telescopic boom, and LMI sensors certified.`;
      }
    }
  }

  // --- Device Pairing Simulation ---
  function markDevicePairedUI() {
    const deviceStatusBadge = document.getElementById('deviceStatusBadge');
    const deviceStatusText = document.getElementById('deviceStatusText');
    const simulatePairBtn = document.getElementById('simulatePairBtn');

    const indicator = deviceStatusBadge ? deviceStatusBadge.querySelector('.status-indicator') : null;
    if (indicator) indicator.classList.add('paired');
    if (deviceStatusText) deviceStatusText.textContent = i18n[currentLocale].pairedStatus;
    if (simulatePairBtn) {
      simulatePairBtn.disabled = true;
      simulatePairBtn.textContent = "✅ Ed25519 Hardware Enclave Bound";
    }
  }

  function simulateDevicePairing() {
    devicePaired = true;
    markDevicePairedUI();
    updateTrustScore();
    saveState();
  }

  // --- Optical Vector Document Layer Zoom Engine ---
  function setDocumentZoom(zoomLevel) {
    currentZoom = Math.min(Math.max(zoomLevel, 0.5), 2.0);
    const certZoomWrapper = document.getElementById('certZoomWrapper');
    const certZoomValue = document.getElementById('certZoomValue');
    
    if (certZoomWrapper) {
      if ('zoom' in certZoomWrapper.style) {
        certZoomWrapper.style.zoom = currentZoom;
        certZoomWrapper.style.transform = 'none';
      } else {
        certZoomWrapper.style.transform = `scale(${currentZoom})`;
        certZoomWrapper.style.transformOrigin = 'top center';
      }
    }
    if (certZoomValue) {
      certZoomValue.textContent = `${Math.round(currentZoom * 100)}%`;
    }
  }

  // --- Real-time Dynamic Certificate Generator ---
  function generateVerifiedCertificate() {
    const rawPattern = document.getElementById('certPattern')?.value || DEFAULT_PATTERN;
    const certNumber = parsePattern(rawPattern, "CRN", 1);
    const company = document.getElementById('companyName')?.value || "Apex Industrial Inspection Services LLC";
    const certRenderBody = document.getElementById('certRenderBody');
    const certModal = document.getElementById('certModal');

    const isRTL = currentLocale === 'ar';
    const isOverload = (appliedLoad > 115);
    const isRopeFailed = (wireRopeStatus === "BROKEN_WIRES" || wireRopeStatus === "CORE_COLLAPSE");
    const isFailed = (isOverload || isRopeFailed);

    const certHTML = `
      <div class="cert-sheet ${isRTL ? 'rtl-mode' : ''}">
        <!-- Watermark Seal -->
        <svg class="cert-watermark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>

        <!-- Top Header Bar -->
        <div class="cert-top-bar">
          <div class="cert-company-heading">
            <h2>${company}</h2>
            <p class="cert-company-sub">Accredited Inspection Body • ISO/IEC 17020 Type A • LEEA Accredited</p>
            <div class="cert-title-badge ${isFailed ? 'badge-failed' : ''}">
              ${isFailed 
                ? (isRTL ? 'تقرير فحص هندسي: غير صالح للتشغيل (قيد الحجر الفني)' : 'INSPECTION CERTIFICATE: FAILED / QUARANTINED')
                : (isRTL ? 'شهادة الفحص الشامل واختبار الحمل الدوري' : 'CERTIFICATE OF THOROUGH EXAMINATION & LOAD TEST')}
            </div>
          </div>
          <div class="cert-qr-panel">
            <svg width="56" height="56" viewBox="0 0 100 100" fill="#0F172A">
              <path d="M0 0h30v30H0zM5 5h20v20H5zM10 10h10v10H10zM70 0h30v30H70zM75 5h20v20H75zM80 10h10v10H80zM0 70h30v30H0zM5 75h20v20H5zM10 80h10v10H10zM35 10h10v10H35zM50 10h10v10H50zM35 25h10v10H35zM50 25h10v10H50zM10 35h10v10H10zM25 35h10v10H25zM10 50h10v10H10zM25 50h10v10H25zM35 70h10v10H35zM50 70h10v10H50zM35 85h10v10H35zM50 85h10v10H50zM70 35h10v10H70zM85 35h10v10H85zM70 50h10v10H70zM85 50h10v10H85zM70 70h30v30H70zM75 75h20v20H75zM80 80h10v10H80z"/>
            </svg>
            <span>CRYPTOGRAPHIC SEAL</span>
          </div>
        </div>

        <!-- Technical Metadata Grid Table -->
        <div class="cert-meta-grid">
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'رقم الشهادة الرسمي' : 'Certificate Number'}</span>
            <span class="cert-meta-value highlight">${certNumber}</span>
          </div>
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'تاريخ الفحص / الاستحقاق' : 'Inspection Date / Next Due'}</span>
            <span class="cert-meta-value">2026-09-02 • ${isFailed ? (isRTL ? 'موقوف فورياً' : 'SUSPENDED') : 'Due: 2027-03-02'}</span>
          </div>
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'وصف المعدة والموديل' : 'Equipment & Model'}</span>
            <span class="cert-meta-value">Mobile Hydraulic Crane • Kato NK-1000</span>
          </div>
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'الرقم التسلسلي / الشاسيه' : 'Serial / Asset Tag'}</span>
            <span class="cert-meta-value">NK1000-88421 (Asset #CR-001)</span>
          </div>
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'الحمل الاسمي / حمل الإثبات المطبق' : 'Rated SWL / Applied Proof Load'}</span>
            <span class="cert-meta-value">100.0T SWL • <strong>${appliedLoad}.0T Applied Load</strong></span>
          </div>
          <div class="cert-meta-item">
            <span class="cert-meta-label">${isRTL ? 'حالة حبل السلك (ISO 4309)' : 'Wire Rope Assessment (ISO 4309)'}</span>
            <span class="cert-meta-value ${isRopeFailed ? 'text-danger' : ''}">${wireRopeStatus.replace('_', ' ')}</span>
          </div>
          <div class="cert-meta-item" style="grid-column: span 2;">
            <span class="cert-meta-label">${isRTL ? 'العميل وموقع الفحص الميداني' : 'Client Organization & Location'}</span>
            <span class="cert-meta-value">Al-Sharq Petrochemicals (Jubail Industrial Plant Yard 4)</span>
          </div>
        </div>

        <!-- Dynamic Result Banner -->
        <div class="cert-result-banner ${isFailed ? 'banner-failed' : ''}">
          <div class="cert-result-header">
            <span class="cert-result-tag ${isFailed ? 'tag-failed' : ''}">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="${isFailed ? '18 6 6 18' : '20 6 9 17 4 12'}"/><line x1="${isFailed ? '6' : '0'}" y1="${isFailed ? '6' : '0'}" x2="${isFailed ? '18' : '0'}" y2="${isFailed ? '18' : '0'}"/></svg>
              ${isFailed 
                ? (isRTL ? 'النتيجة الفنية: راسب - غير صالح للاستخدام (QUARANTINED)' : 'TECHNICAL RESULT: FAILED - UNSAFE FOR OPERATION')
                : (isRTL ? 'النتيجة الفنية: صالح للتشغيل وآمن للاستخدام (PASSED)' : 'TECHNICAL RESULT: PASSED & SAFE FOR OPERATION')}
            </span>
          </div>
          <p class="cert-result-text">
            ${isFailed
              ? (isOverload 
                  ? (isRTL ? 'تم إلغاء الشهادة بسبب تطبيق حمل زائد يتجاوز معايير السلامة الهندسية (ASME B30.5). المعدة غير آمنة.' : 'Certificate rejected due to structural overload testing outside allowable engineering envelopes under ASME B30.5.')
                  : (isRTL ? 'تم إلغاء الشهادة بسبب تجاوز حبل السلك حدود التآكل الحرجة ومعايير الاستبعاد الإلزامية وفق ISO 4309.' : 'Certificate rejected due to severe wire rope defect scoring meeting mandatory discard criteria under ISO 4309.'))
              : (isRTL 
                  ? `تم اختبار الحمل بنجاح عند (${appliedLoad} طن) دون أي انحراف دائم. حبل السلك يقع ضمن الحدود المسموح بها وفق معيار ISO 4309.` 
                  : `Outrigger hydraulics, telescopic boom integrity, and proof load test of ${appliedLoad}T verified under BS 7121 / ISO 4309.`)}
          </p>
        </div>

        <!-- Dual Signature & Crypto Verification Bar -->
        <div class="cert-signatures-bar">
          <div class="cert-sign-block">
            <span class="cert-sign-title">${isRTL ? 'فاحص الميدان المعتمد (LEEA)' : 'Qualified Field Inspector'}</span>
            <span class="cert-sign-name">Tariq Mansoor (LEEA Cert #88319)</span>
            <span class="cert-crypto-hash">Ed25519 Sig: 7f8a9e...421c</span>
          </div>
          <div class="cert-sign-block" style="${isRTL ? 'text-align: left;' : 'text-align: right;'}">
            <span class="cert-sign-title">${isRTL ? 'المدير الفني المعتمد (QA/QC)' : 'Technical Authority (Four-Eyes)'}</span>
            <span class="cert-sign-name">Eng. Ahmed Al-Ghamdi</span>
            <span class="cert-crypto-hash">Official Seal: APEX-STAMP-2026-QA</span>
          </div>
        </div>

        <!-- Accreditations & Badges Footer -->
        <div class="cert-accreditations-footer">
          <div class="cert-badge-pills">
            <span class="badge-pill pill-leea">LEEA MEMBER #4921</span>
            <span class="badge-pill pill-iso">ISO/IEC 17020</span>
            <span class="badge-pill pill-zatca">ZATCA READY</span>
          </div>
          <span class="cert-security-seal">INTEGIN PKI VERIFIED • TAMPER-PROOF</span>
        </div>
      </div>
    `;

    if (certRenderBody) certRenderBody.innerHTML = certHTML;
    setDocumentZoom(1.0);
    if (certModal) certModal.classList.add('open');
  }

  // --- Localization Switching ---
  function applyLanguage(lang) {
    currentLocale = lang;
    const isArabic = (lang === 'ar');

    document.documentElement.lang = lang;
    document.documentElement.dir = isArabic ? 'rtl' : 'ltr';
    document.body.classList.toggle('rtl-mode', isArabic);

    const langText = document.getElementById('langText');
    if (langText) langText.textContent = isArabic ? 'English (LTR)' : 'العربية (RTL)';

    document.querySelectorAll('[data-i18n]').forEach(el => {
      const key = el.getAttribute('data-i18n');
      if (i18n[lang] && i18n[lang][key]) {
        el.textContent = i18n[lang][key];
      }
    });

    renderStep(currentStep);
    updatePhysicsVerdict();
  }

  // --- Event Bindings ---
  function bindEvents() {
    // Stepper Nav Buttons
    document.querySelectorAll('.step-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const targetStep = parseInt(btn.getAttribute('data-step'), 10);
        renderStep(targetStep);
      });
    });

    // Next / Prev Buttons
    const nextStepBtn = document.getElementById('nextStepBtn');
    const prevStepBtn = document.getElementById('prevStepBtn');
    if (nextStepBtn) {
      nextStepBtn.addEventListener('click', () => {
        if (currentStep < totalSteps) {
          renderStep(currentStep + 1);
        } else {
          alert("🎉 Onboarding Completed! Welcome to INTEGIN Production Dashboard.");
        }
      });
    }

    if (prevStepBtn) {
      prevStepBtn.addEventListener('click', () => {
        if (currentStep > 1) renderStep(currentStep - 1);
      });
    }

    // Language Toggle
    const langToggleBtn = document.getElementById('langToggleBtn');
    if (langToggleBtn) {
      langToggleBtn.addEventListener('click', () => {
        applyLanguage(currentLocale === 'en' ? 'ar' : 'en');
      });
    }

    // Signing Policy Radio Selection
    document.querySelectorAll('.policy-card').forEach(card => {
      card.addEventListener('click', () => {
        const radio = card.querySelector('input[name="signPolicy"]');
        if (radio) {
          radio.checked = true;
          state.signPolicy = radio.value;
          document.querySelectorAll('.policy-card').forEach(c => c.classList.remove('selected'));
          card.classList.add('selected');
          saveState();
        }
      });
    });

    // Discipline Card Selection
    document.querySelectorAll('.discipline-card').forEach(card => {
      card.addEventListener('click', () => {
        const cb = card.querySelector('input[name="discipline"]');
        if (cb) {
          cb.checked = !cb.checked;
          card.classList.toggle('active', cb.checked);
          saveState();
        }
      });
    });

    // Accreditation Checkboxes
    document.querySelectorAll('.badge-checkbox-card input[type="checkbox"]').forEach(cb => {
      cb.addEventListener('change', saveState);
    });

    // Smart Pattern Input & Tag Click Handlers
    const certPatternInput = document.getElementById('certPattern');
    if (certPatternInput) {
      certPatternInput.addEventListener('input', () => {
        updateCertificatePreview();
        saveState();
      });
    }

    // Dynamic Tag Chip Insertion Buttons
    document.querySelectorAll('.tag-chip-btn[data-insert]').forEach(btn => {
      btn.addEventListener('click', () => {
        const tagToInsert = btn.getAttribute('data-insert');
        if (certPatternInput) {
          const start = certPatternInput.selectionStart || certPatternInput.value.length;
          const end = certPatternInput.selectionEnd || certPatternInput.value.length;
          const currentVal = certPatternInput.value;
          
          certPatternInput.value = currentVal.substring(0, start) + tagToInsert + currentVal.substring(end);
          certPatternInput.focus();
          certPatternInput.selectionStart = certPatternInput.selectionEnd = start + tagToInsert.length;
          
          updateCertificatePreview();
          saveState();
        }
      });
    });

    // Reset Pattern Button
    const resetPatternBtn = document.getElementById('resetPatternBtn');
    if (resetPatternBtn && certPatternInput) {
      resetPatternBtn.addEventListener('click', () => {
        certPatternInput.value = DEFAULT_PATTERN;
        updateCertificatePreview();
        saveState();
      });
    }

    // Subdomain change updates prefix
    const subdomainSlug = document.getElementById('subdomainSlug');
    if (subdomainSlug) {
      subdomainSlug.addEventListener('input', () => {
        updateCertificatePreview();
        saveState();
      });
    }

    // Physics Slider: Applied Load
    const loadRange = document.getElementById('loadRange');
    if (loadRange) {
      loadRange.addEventListener('input', (e) => {
        appliedLoad = parseInt(e.target.value, 10);
        updatePhysicsVerdict();
      });
    }

    // Physics Select: Wire Rope Evaluation
    const wireRopeSelect = document.getElementById('wireRopeSelect');
    if (wireRopeSelect) {
      wireRopeSelect.addEventListener('change', (e) => {
        wireRopeStatus = e.target.value;
        updatePhysicsVerdict();
      });
    }

    // Simulate Device Pairing
    const simulatePairBtn = document.getElementById('simulatePairBtn');
    if (simulatePairBtn) {
      simulatePairBtn.addEventListener('click', simulateDevicePairing);
    }

    // Certificate Modal Actions
    const generateCertBtn = document.getElementById('generateCertBtn');
    const closeCertModalBtn = document.getElementById('closeCertModalBtn');
    const certModal = document.getElementById('certModal');
    const finishOnboardingBtn = document.getElementById('finishOnboardingBtn');

    if (generateCertBtn) {
      generateCertBtn.addEventListener('click', generateVerifiedCertificate);
    }

    if (closeCertModalBtn && certModal) {
      closeCertModalBtn.addEventListener('click', () => {
        certModal.classList.remove('open');
      });
    }

    if (finishOnboardingBtn && certModal) {
      finishOnboardingBtn.addEventListener('click', () => {
        certModal.classList.remove('open');
        alert("🚀 Tenant Provisioned & Activated for Production!");
      });
    }

    // Document Zoom Handlers
    const certZoomInBtn = document.getElementById('certZoomInBtn');
    const certZoomOutBtn = document.getElementById('certZoomOutBtn');
    const certZoomResetBtn = document.getElementById('certZoomResetBtn');

    if (certZoomInBtn) {
      certZoomInBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        setDocumentZoom(currentZoom + 0.15);
      });
    }

    if (certZoomOutBtn) {
      certZoomOutBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        setDocumentZoom(currentZoom - 0.15);
      });
    }

    if (certZoomResetBtn) {
      certZoomResetBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        setDocumentZoom(1.0);
      });
    }

    // Mousewheel + Ctrl Zoom
    const certModalBody = document.getElementById('certModalBody');
    if (certModalBody) {
      certModalBody.addEventListener('wheel', (e) => {
        if (e.ctrlKey) {
          e.preventDefault();
          const delta = e.deltaY < 0 ? 0.08 : -0.08;
          setDocumentZoom(currentZoom + delta);
        }
      }, { passive: false });
    }

    // Reset Setup
    const resetWizardBtn = document.getElementById('resetWizardBtn');
    if (resetWizardBtn) {
      resetWizardBtn.addEventListener('click', () => {
        if (confirm("Reset setup back to Step 1 and wipe draft data?")) {
          localStorage.removeItem(STORAGE_KEY);
          location.reload();
        }
      });
    }

    // Crash Simulation
    const simulateCrashBtn = document.getElementById('simulateCrashBtn');
    if (simulateCrashBtn) {
      simulateCrashBtn.addEventListener('click', () => {
        saveState();
        document.body.style.opacity = '0.3';
        document.body.style.pointerEvents = 'none';
        setTimeout(() => {
          location.reload();
        }, 600);
      });
    }

    // Auto-save on inputs
    document.querySelectorAll('.form-control').forEach(input => {
      input.addEventListener('input', saveState);
    });
  }

  document.addEventListener('DOMContentLoaded', init);
})();
