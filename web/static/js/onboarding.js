// Investo Onboarding Tour
(function() {
    'use strict';

    if (localStorage.getItem('investo_tour_completed')) return;

    const steps = [
        {
            selector: null,
            title: 'Ini dashboard Anda',
            description: 'Ringkasan pasar dan portofolio Anda dalam satu layar. Pantau pergerakan saham, berita terkini, dan kondisi pasar real-time.',
            position: 'center'
        },
        {
            selector: 'a[href="/saham"]',
            title: 'Di sini Saham IDX',
            description: 'Cari dan filter 54+ saham Bursa Efek Indonesia. Lihat harga real-time, chart teknikal, dan analisa fundamental.',
            position: 'bottom'
        },
        {
            selector: 'a[href="/screener"]',
            title: 'Screener canggih',
            description: 'Filter saham dengan 15+ kriteria: PER, PBV, ROE, market cap, sektor, dan lainnya. Temukan saham undervalue dalam hitungan detik.',
            position: 'bottom'
        },
        {
            selector: 'a[href="/dashboard/portfolios"]',
            title: 'Buat portofolio pertama Anda',
            description: 'Track investasi Anda secara real-time. Catat setiap transaksi beli/jual, pantau profit/loss, dan analisa kinerja portofolio.',
            position: 'bottom'
        },
        {
            selector: 'a[href="/dashboard/alerts"]',
            title: 'Alert harga',
            description: 'Atur alert pada harga saham tertentu. Dapatkan notifikasi saat harga mencapai target, support/resistance tembus, atau pola teknikal terdeteksi.',
            position: 'bottom'
        }
    ];

    let currentStep = -1;
    let overlay, tooltip, highlight;

    function createElements() {
        overlay = document.createElement('div');
        overlay.className = 'tour-overlay';
        overlay.style.cssText = 'position:fixed;inset:0;z-index:9999;background:rgba(0,0,0,0.65);transition:opacity 0.3s;';

        tooltip = document.createElement('div');
        tooltip.className = 'tour-tooltip';
        tooltip.style.cssText = 'position:fixed;z-index:10001;background:#fff;border-radius:16px;box-shadow:0 20px 60px rgba(0,0,0,0.3);padding:24px;max-width:380px;width:calc(100vw - 48px);transition:all 0.35s cubic-bezier(0.16,1,0.3,1);font-family:Inter,system-ui,sans-serif;';

        document.body.appendChild(overlay);
        document.body.appendChild(tooltip);
    }

    function showStep(stepIndex) {
        if (stepIndex >= steps.length) {
            finish();
            return;
        }

        currentStep = stepIndex;
        const step = steps[currentStep];

        const target = step.selector ? document.querySelector(step.selector) : null;

        if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'center' });
            setTimeout(() => positionTooltip(target, step), 400);
        } else {
            positionCenter(step);
        }

        renderTooltipContent(step, stepIndex);
    }

    function positionTooltip(target, step) {
        const rect = target.getBoundingClientRect();
        const tooltipW = 380;
        const tooltipH = tooltip.offsetHeight || 200;
        let top, left;

        switch (step.position) {
            case 'bottom':
                top = rect.bottom + 12;
                left = rect.left + rect.width / 2 - tooltipW / 2;
                break;
            case 'top':
                top = rect.top - tooltipH - 12;
                left = rect.left + rect.width / 2 - tooltipW / 2;
                break;
            case 'left':
                top = rect.top + rect.height / 2 - tooltipH / 2;
                left = rect.left - tooltipW - 12;
                break;
            case 'right':
                top = rect.top + rect.height / 2 - tooltipH / 2;
                left = rect.right + 12;
                break;
            default:
                top = rect.bottom + 12;
                left = rect.left + rect.width / 2 - tooltipW / 2;
        }

        left = Math.max(12, Math.min(window.innerWidth - tooltipW - 12, left));
        top = Math.max(12, Math.min(window.innerHeight - tooltipH - 12, top));

        tooltip.style.top = top + 'px';
        tooltip.style.left = left + 'px';
        tooltip.style.opacity = '1';
        tooltip.style.transform = 'scale(1)';

        target.style.position = 'relative';
        target.style.zIndex = '10000';
        target.style.boxShadow = '0 0 0 4px #3b82f6, 0 0 0 8px rgba(59,130,246,0.25)';
        target.style.borderRadius = '8px';
        target.style.transition = 'box-shadow 0.3s';

        if (highlight) {
            highlight.style.boxShadow = '';
            highlight.style.zIndex = '';
            highlight.style.borderRadius = '';
        }
        highlight = target;
    }

    function positionCenter(step) {
        tooltip.style.top = '50%';
        tooltip.style.left = '50%';
        tooltip.style.transform = 'translate(-50%, -50%)';
        tooltip.style.opacity = '1';
    }

    function renderTooltipContent(step, stepIndex) {
        const total = steps.length;
        const isLast = stepIndex === total - 1;

        tooltip.innerHTML = `
            <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;">
                <span style="font-size:11px;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px;">Langkah ${stepIndex + 1} dari ${total}</span>
                <button onclick="document.querySelector('.tour-skip').click()" style="background:none;border:none;color:#94a3b8;cursor:pointer;font-size:18px;line-height:1;padding:4px;">&times;</button>
            </div>
            <h3 style="font-size:18px;font-weight:700;color:#0f172a;margin:0 0 8px;font-family:'Playfair Display',Georgia,serif;">${step.title}</h3>
            <p style="font-size:14px;color:#475569;line-height:1.65;margin:0 0 20px;">${step.description}</p>
            <div style="display:flex;align-items:center;justify-content:space-between;">
                <button class="tour-skip" onclick="window.__tourSkip()" style="background:none;border:none;color:#94a3b8;font-size:13px;cursor:pointer;padding:8px 12px;border-radius:8px;font-weight:500;">${isLast ? 'Selesai' : 'Lewati'}</button>
                <button class="tour-next" onclick="window.__tourNext()" style="background:linear-gradient(135deg,#2563eb,#1e40af);color:#fff;border:none;font-size:13px;font-weight:600;cursor:pointer;padding:10px 20px;border-radius:10px;box-shadow:0 4px 12px rgba(37,99,235,0.3);">
                    ${isLast ? 'Selesai' : 'Selanjutnya →'}
                </button>
            </div>
        `;
    }

    function next() {
        showStep(currentStep + 1);
    }

    function finish() {
        if (highlight) {
            highlight.style.boxShadow = '';
            highlight.style.zIndex = '';
            highlight.style.borderRadius = '';
        }
        overlay.remove();
        tooltip.remove();
        localStorage.setItem('investo_tour_completed', 'true');
    }

    window.__tourNext = next;
    window.__tourSkip = finish;

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    function init() {
        // Only show on dashboard
        if (!window.location.pathname.match(/^\/dashboard\/?$/)) {
            // Still mark as seen if user navigates away
            if (localStorage.getItem('investo_tour_completed')) return;
            return;
        }
        createElements();
        showStep(0);
    }
})();
