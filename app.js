async function fetchLifts() {
    try {
        const response = await fetch('data/latest.json');
        if (!response.ok) throw new Error('Data not found');
        const data = await response.json();

        if (data.last_updated && document.getElementById('last-updated')) {
            document.getElementById('last-updated').innerText = `Last updated: ${data.last_updated}`;
        }

        renderLifts(data.lifts || []);
    } catch (error) {
        console.error('Error fetching lifts:', error);
        const statusEl = document.getElementById('today-status');
        if (statusEl) statusEl.innerText = 'Error';
        const detailsEl = document.getElementById('today-details');
        if (detailsEl) detailsEl.innerText = 'Could not load lift schedule.';
    } finally {
        const loader = document.querySelector('.loader');
        if (loader) loader.style.display = 'none';
    }
}

function generateICS(lift) {
    const [d, m, y] = lift.date_dmy.split('/').map(Number);
    const [hours, minutes] = lift.time.split(':').map(Number);

    const startDate = new Date(y, m - 1, d, hours, minutes);
    const endDate = new Date(startDate.getTime() + 15 * 60000);

    const formatDateICS = (date) => {
        return date.toISOString().replace(/[-:]/g, '').split('.')[0] + 'Z';
    };

    const sourceInfo = "Schedule provided by Tower Bridge Lifts Project (https://surajnarwade.github.com/towerbridgelifts/)";
    const icsContent = [
        'BEGIN:VCALENDAR',
        'VERSION:2.0',
        'PRODID:-//Tower Bridge Lifts//EN',
        'BEGIN:VEVENT',
        `UID:${Date.now()}@towerbridge.org.uk`,
        `DTSTAMP:${formatDateICS(new Date())}`,
        `DTSTART:${formatDateICS(startDate)}`,
        `DTEND:${formatDateICS(endDate)}`,
        `SUMMARY:Tower Bridge Lift - ${lift.vessel}`,
        `DESCRIPTION:Vessel: ${lift.vessel}\\nType: ${lift.vessel_type}\\nDirection: ${lift.direction}\\n\\n${sourceInfo}`,
        'LOCATION:Tower Bridge, London',
        'END:VEVENT',
        'END:VCALENDAR'
    ].join('\r\n');

    const blob = new Blob([icsContent], { type: 'text/calendar;charset=utf-8' });
    const link = document.createElement('a');
    link.href = window.URL.createObjectURL(blob);
    link.download = `lift_${lift.vessel.replace(/\s+/g, '_')}_${lift.date_dmy.replace(/\//g, '')}.ics`;
    link.click();
}

function openGoogleCalendar(lift) {
    const [d, m, y] = lift.date_dmy.split('/').map(Number);
    const [hours, minutes] = lift.time.split(':').map(Number);

    const startDate = new Date(y, m - 1, d, hours, minutes);
    const endDate = new Date(startDate.getTime() + 15 * 60000);

    const formatDateGCal = (date) => {
        return date.toISOString().replace(/[-:]/g, '').split('.')[0] + 'Z';
    };

    const sourceInfo = "Schedule provided by Tower Bridge Lifts Project (https://surajnarwade.github.com/towerbridgelifts/)";
    const title = encodeURIComponent(`Tower Bridge Lift - ${lift.vessel}`);
    const dates = `${formatDateGCal(startDate)}/${formatDateGCal(endDate)}`;
    const details = encodeURIComponent(`Vessel: ${lift.vessel}\nType: ${lift.vessel_type}\nDirection: ${lift.direction}\n\n${sourceInfo}`);
    const location = encodeURIComponent('Tower Bridge, London');

    const url = `https://calendar.google.com/calendar/r/eventedit?action=TEMPLATE&text=${title}&dates=${dates}&details=${details}&location=${location}`;
    window.open(url, '_blank');
}

function renderLifts(lifts) {
    const today = new Date();
    const todayStr = formatDate(today);

    const liftsToday = lifts.filter(lift => lift.date_dmy === todayStr);

    const statusEl = document.getElementById('today-status');
    const detailsEl = document.getElementById('today-details');

    if (statusEl && detailsEl) {
        if (liftsToday.length > 0) {
            statusEl.innerText = 'YES';
            statusEl.className = 'status-value status-yes';
            detailsEl.innerHTML = `<p>${liftsToday.length} lift(s) scheduled for today.</p>`;

            const btnContainer = document.createElement('div');
            btnContainer.className = 'btn-group-main';

            liftsToday.forEach(l => {
                const group = document.createElement('div');
                group.className = 'btn-group-item';

                const btnICS = document.createElement('button');
                btnICS.className = 'btn-add-calendar';
                btnICS.innerText = `Add to iCal / Outlook (${l.time})`;
                btnICS.onclick = () => generateICS(l);

                const btnGCal = document.createElement('button');
                btnGCal.className = 'btn-google-calendar';
                btnGCal.innerText = `Add to Google Calendar (${l.time})`;
                btnGCal.onclick = () => openGoogleCalendar(l);

                group.appendChild(btnICS);
                group.appendChild(btnGCal);
                btnContainer.appendChild(group);
            });
            detailsEl.appendChild(btnContainer);
        } else {
            statusEl.innerText = 'NO';
            statusEl.className = 'status-value status-no';
            detailsEl.innerText = 'No lifts scheduled for today.';
        }
    }

    const upcomingList = document.getElementById('lifts-list');
    if (upcomingList) {
        upcomingList.innerHTML = '';

        const futureLifts = lifts.filter(lift => {
            const [d, m, y] = lift.date_dmy.split('/').map(Number);
            const [h, min] = lift.time.split(':').map(Number);
            const liftDate = new Date(y, m - 1, d, h, min);
            return liftDate >= today;
        });

        futureLifts.slice(0, 12).forEach(lift => {
            const div = document.createElement('div');
            div.className = 'lift-item';
            div.innerHTML = `
                <div class="lift-date">${lift.full_date}</div>
                <div class="lift-time">${lift.time}</div>
                <div class="lift-vessel">${lift.vessel}</div>
                <div class="lift-info">${lift.direction} • ${lift.vessel_type}</div>
                <div class="btn-group-small">
                    <button class="btn-add-calendar-small" title="Add to iCal / Outlook">iCal / Outlook</button>
                    <button class="btn-google-calendar-small" title="Add to Google Calendar">Google Calendar</button>
                </div>
            `;
            div.querySelector('.btn-add-calendar-small').onclick = () => generateICS(lift);
            div.querySelector('.btn-google-calendar-small').onclick = () => openGoogleCalendar(lift);
            upcomingList.appendChild(div);
        });
    }
}

function formatDate(date) {
    const d = String(date.getDate()).padStart(2, '0');
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const y = date.getFullYear();
    return `${d}/${m}/${y}`;
}

// Theme Toggle Logic
function initTheme() {
    const themeSwitch = document.querySelector('.theme-switch input[type="checkbox"]');
    const body = document.body;

    // Load saved theme
    const savedTheme = localStorage.getItem('theme') || 'dark-mode';
    body.className = savedTheme;

    // Sync checkbox state
    if (themeSwitch) {
        themeSwitch.checked = (savedTheme === 'light-mode');

        themeSwitch.addEventListener('change', (e) => {
            if (e.target.checked) {
                body.className = 'light-mode';
                localStorage.setItem('theme', 'light-mode');
            } else {
                body.className = 'dark-mode';
                localStorage.setItem('theme', 'dark-mode');
            }
        });
    }
}

document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    fetchLifts();
});
