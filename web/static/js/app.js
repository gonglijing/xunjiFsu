// ============================================
// HuShu智能网关 - 统一前端应用脚本
// ============================================

// 工具函数
const Utils = {
    // 格式化时间
    formatTime(date) {
        return new Date(date).toLocaleString('zh-CN');
    },
    
    // 获取单位
    getUnit(fieldName) {
        fieldName = (fieldName || '').toLowerCase();
        if (fieldName.includes('temp') || fieldName.includes('温度')) return '°C';
        if (fieldName.includes('hum') || fieldName.includes('湿度')) return '%';
        if (fieldName.includes('press') || fieldName.includes('压力') || fieldName.includes('压强')) return 'Pa';
        if (fieldName.includes('volt') || fieldName.includes('电压')) return 'V';
        if (fieldName.includes('curr') || fieldName.includes('电流')) return 'A';
        if (fieldName.includes('power') || fieldName.includes('功率')) return 'W';
        if (fieldName.includes('speed') || fieldName.includes('速度')) return 'm/s';
        if (fieldName.includes('level') || fieldName.includes('液位')) return 'm';
        if (fieldName.includes('flow') || fieldName.includes('流量')) return 'm³/h';
        return '';
    },
    
    // 格式化数值
    formatValue(value) {
        if (value === null || value === undefined) return '--';
        const num = parseFloat(value);
        return isNaN(num) ? value : num.toFixed(2);
    },
    
    // 生成唯一ID
    generateId() {
        return Date.now().toString(36) + Math.random().toString(36).substr(2);
    },
    
    // API请求
    async api(url, options = {}) {
        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            return await response.json();
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }
};

// 粒子背景效果
class ParticleBackground {
    constructor() {
        this.canvas = null;
        this.ctx = null;
        this.particles = [];
        this.animationId = null;
    }
    
    init(canvasId = 'particle-canvas') {
        const existing = document.getElementById(canvasId);
        if (existing) existing.remove();
        
        this.canvas = document.createElement('canvas');
        this.canvas.id = canvasId;
        this.canvas.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            z-index: -1;
            pointer-events: none;
            opacity: 0.6;
        `;
        document.body.appendChild(this.canvas);
        
        this.ctx = this.canvas.getContext('2d');
        this.resize();
        
        window.addEventListener('resize', () => this.resize());
        
        // 创建粒子
        for (let i = 0; i < 100; i++) {
            this.particles.push(this.createParticle());
        }
        
        this.animate();
    }
    
    createParticle() {
        return {
            x: Math.random() * this.canvas.width,
            y: Math.random() * this.canvas.height,
            size: Math.random() * 2 + 0.5,
            speedX: (Math.random() - 0.5) * 0.5,
            speedY: (Math.random() - 0.5) * 0.5,
            opacity: Math.random() * 0.5 + 0.2
        };
    }
    
    resize() {
        this.canvas.width = window.innerWidth;
        this.canvas.height = window.innerHeight;
    }
    
    animate() {
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        
        this.particles.forEach(p => {
            p.x += p.speedX;
            p.y += p.speedY;
            
            if (p.x < 0 || p.x > this.canvas.width) p.speedX *= -1;
            if (p.y < 0 || p.y > this.canvas.height) p.speedY *= -1;
            
            this.ctx.beginPath();
            this.ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
            this.ctx.fillStyle = `rgba(0, 198, 255, ${p.opacity})`;
            this.ctx.fill();
        });
        
        // 绘制连接线
        this.particles.forEach((p1, i) => {
            this.particles.slice(i + 1).forEach(p2 => {
                const dx = p1.x - p2.x;
                const dy = p1.y - p2.y;
                const dist = Math.sqrt(dx * dx + dy * dy);
                
                if (dist < 150) {
                    this.ctx.beginPath();
                    this.ctx.moveTo(p1.x, p1.y);
                    this.ctx.lineTo(p2.x, p2.y);
                    this.ctx.strokeStyle = `rgba(0, 198, 255, ${0.1 * (1 - dist / 150)})`;
                    this.ctx.stroke();
                }
            });
        });
        
        this.animationId = requestAnimationFrame(() => this.animate());
    }
    
    destroy() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
        if (this.canvas) {
            this.canvas.remove();
        }
    }
}

// 数字滚动动画
class NumberCounter {
    constructor(element, target, duration = 1000) {
        this.element = element;
        this.target = target;
        this.duration = duration;
    }
    
    start() {
        const start = parseFloat(this.element.textContent) || 0;
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / this.duration, 1);
            
            const easeOut = 1 - Math.pow(1 - progress, 3);
            const current = start + (this.target - start) * easeOut;
            
            this.element.textContent = Math.round(current);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }
}

// 图表渲染器 (简化版)
class ChartRenderer {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        this.options = {
            type: 'line',
            color: '#00c6ff',
            height: 200,
            ...options
        };
        this.data = [];
    }
    
    setData(data) {
        this.data = data;
        this.render();
    }
    
    render() {
        if (!this.container || this.data.length === 0) return;
        
        const canvas = document.createElement('canvas');
        canvas.width = this.container.clientWidth || 400;
        canvas.height = this.options.height;
        canvas.style.width = '100%';
        canvas.style.height = this.options.height + 'px';
        
        const ctx = canvas.getContext('2d');
        const width = canvas.width;
        const height = canvas.height;
        const padding = 20;
        
        // 计算数据范围
        const values = this.data.map(d => d.value);
        const min = Math.min(...values) * 0.9;
        const max = Math.max(...values) * 1.1;
        const range = max - min || 1;
        
        // 绘制网格
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)';
        ctx.lineWidth = 1;
        for (let i = 0; i <= 4; i++) {
            const y = padding + (height - 2 * padding) * (i / 4);
            ctx.beginPath();
            ctx.moveTo(padding, y);
            ctx.lineTo(width - padding, y);
            ctx.stroke();
        }
        
        // 绘制数据线
        const gradient = ctx.createLinearGradient(0, 0, 0, height);
        gradient.addColorStop(0, this.options.color + '80');
        gradient.addColorStop(1, this.options.color + '00');
        
        ctx.beginPath();
        ctx.moveTo(padding, height - padding);
        
        this.data.forEach((d, i) => {
            const x = padding + (width - 2 * padding) * (i / (this.data.length - 1));
            const y = padding + (height - 2 * padding) * (1 - (d.value - min) / range);
            ctx.lineTo(x, y);
        });
        
        ctx.strokeStyle = this.options.color;
        ctx.lineWidth = 2;
        ctx.stroke();
        
        // 填充区域
        ctx.lineTo(width - padding, height - padding);
        ctx.closePath();
        ctx.fillStyle = gradient;
        ctx.fill();
        
        // 绘制数据点
        this.data.forEach((d, i) => {
            const x = padding + (width - 2 * padding) * (i / (this.data.length - 1));
            const y = padding + (height - 2 * padding) * (1 - (d.value - min) / range);
            
            ctx.beginPath();
            ctx.arc(x, y, 4, 0, Math.PI * 2);
            ctx.fillStyle = this.options.color;
            ctx.fill();
            
            ctx.beginPath();
            ctx.arc(x, y, 6, 0, Math.PI * 2);
            ctx.strokeStyle = this.options.color + '40';
            ctx.lineWidth = 2;
            ctx.stroke();
        });
        
        this.container.innerHTML = '';
        this.container.appendChild(canvas);
    }
}

// 模态框管理器
class ModalManager {
    constructor() {
        this.stack = [];
    }
    
    show(content, options = {}) {
        const container = document.getElementById('modal-container');
        if (!container) return;
        
        container.innerHTML = `
            <div class="modal-overlay absolute inset-0" onclick="Modal.close()"></div>
            <div class="modal-content absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 
                        rounded-2xl p-6 w-full max-w-lg max-h-[80vh] overflow-auto
                        ${options.className || ''}">
                ${content}
            </div>
        `;
        container.classList.remove('hidden');
        
        // 动画
        const modalElement = container.querySelector('.modal-content');
        modalElement.style.opacity = '0';
        modalElement.style.transform = 'translate(-50%, -50%) scale(0.9)';
        
        requestAnimationFrame(() => {
            modalElement.style.transition = 'all 0.3s ease';
            modalElement.style.opacity = '1';
            modalElement.style.transform = 'translate(-50%, -50%) scale(1)';
        });
        
        this.stack.push(container);
    }
    
    close() {
        const container = document.getElementById('modal-container');
        if (container) {
            const modalElement = container.querySelector('.modal-content');
            if (modalElement) {
                modalElement.style.opacity = '0';
                modalElement.style.transform = 'translate(-50%, -50%) scale(0.9)';
            }
            setTimeout(() => {
                container.classList.add('hidden');
                container.innerHTML = '';
            }, 300);
        }
        this.stack.pop();
    }
}

// 标签页管理器
class TabManager {
    constructor() {
        this.tabs = new Map();
    }
    
    register(name, tabBtnId, contentId) {
        this.tabs.set(name, { tabBtnId, contentId });
        
        const btn = document.getElementById(tabBtnId);
        if (btn) {
            btn.addEventListener('click', () => this.switch(name));
        }
    }
    
    switch(name) {
        const tab = this.tabs.get(name);
        if (!tab) return;
        
        // 更新按钮状态
        this.tabs.forEach((t, n) => {
            const btn = document.getElementById(t.tabBtnId);
            if (btn) {
                if (n === name) {
                    btn.classList.add('active');
                } else {
                    btn.classList.remove('active');
                }
            }
            
            const content = document.getElementById(t.contentId);
            if (content) {
                if (n === name) {
                    content.classList.remove('hidden');
                    content.classList.add('fade-in');
                } else {
                    content.classList.add('hidden');
                    content.classList.remove('fade-in');
                }
            }
        });
    }
}

// Toast通知
class Toast {
    static show(message, type = 'info', duration = 3000) {
        const container = document.getElementById('toast-container');
        if (!container) {
            const div = document.createElement('div');
            div.id = 'toast-container';
            div.className = 'fixed bottom-4 right-4 z-50 flex flex-col gap-2';
            document.body.appendChild(div);
        }
        
        const toast = document.createElement('div');
        const colors = {
            success: 'from-green-500 to-emerald-600',
            error: 'from-red-500 to-pink-600',
            warning: 'from-yellow-500 to-orange-600',
            info: 'from-blue-500 to-purple-600'
        };
        
        const icons = {
            success: 'fa-check-circle',
            error: 'fa-exclamation-circle',
            warning: 'fa-exclamation-triangle',
            info: 'fa-info-circle'
        };
        
        toast.className = `bg-gradient-to-r ${colors[type]} text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-in`;
        toast.innerHTML = `
            <i class="fas ${icons[type]}"></i>
            <span>${message}</span>
        `;
        
        document.getElementById('toast-container').appendChild(toast);
        
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }
}

// 初始化全局效果
document.addEventListener('DOMContentLoaded', function() {
    // 初始化粒子背景
    const particleBg = new ParticleBackground();
    particleBg.init();
    
    // 添加全局样式动画
    const style = document.createElement('style');
    style.textContent = `
        @keyframes slide-in {
            from {
                opacity: 0;
                transform: translateX(100%);
            }
            to {
                opacity: 1;
                transform: translateX(0);
            }
        }
        .animate-slide-in {
            animation: slide-in 0.3s ease-out;
        }
        .scan-line {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 2px;
            background: linear-gradient(90deg, transparent, #00c6ff, transparent);
            animation: scan 3s linear infinite;
        }
        @keyframes scan {
            0% { top: 0; opacity: 0; }
            10% { opacity: 1; }
            90% { opacity: 1; }
            100% { top: 100%; opacity: 0; }
        }
        .hologram-effect {
            position: relative;
            overflow: hidden;
        }
        .hologram-effect::after {
            content: '';
            position: absolute;
            inset: 0;
            background: linear-gradient(
                transparent 50%,
                rgba(0, 198, 255, 0.05) 50%,
                transparent 51%
            );
            background-size: 100% 4px;
            pointer-events: none;
        }
    `;
    document.head.appendChild(style);
});

// 导出到全局
window.Utils = Utils;
window.ParticleBackground = ParticleBackground;
window.NumberCounter = NumberCounter;
window.ChartRenderer = ChartRenderer;
window.Modal = new ModalManager();
window.TabManager = new TabManager();
window.Toast = Toast;
