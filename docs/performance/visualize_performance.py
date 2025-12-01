#!/usr/bin/env python3
"""
CKKS Performance Visualization: Native Go vs WASM
Generates comprehensive performance comparison charts
"""

import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import numpy as np
from matplotlib.gridspec import GridSpec

# Set style
plt.style.use('seaborn-v0_8-darkgrid')
colors = {
    'native': '#2ecc71',  # Green
    'wasm': '#e74c3c',     # Red
    'encoding': '#3498db', # Blue
    'encryption': '#e67e22', # Orange
    'other': '#95a5a6'     # Gray
}

# ============================================================================
# Data from actual measurements
# ============================================================================

# Single feature encryption time (ms)
native_time = 0.19
wasm_time = 100.0

# 5 features encryption time (ms)
native_5_features = 0.97
wasm_5_features = 582.0

# WASM breakdown (from detailed timing)
wasm_breakdown = {
    'PK Unmarshal': 1.7,
    'Encoding': 18.7,
    'Encryptor Creation': 2.7,
    'Actual Encryption': 92.8,
    'CT Marshal': 1.0,
    'JS Copy': 0.4
}

# E2E comparison (ms)
e2e_native = {
    'Key Gen': 25,
    'Encrypt (5)': 0.97,
    'Backend': 450,
    'Decrypt': 4
}

e2e_wasm = {
    'Key Gen': 400,
    'Encrypt (5)': 582,
    'Backend': 450,
    'Decrypt': 60
}

# ============================================================================
# Figure 1: Single Feature Comparison (Bar Chart)
# ============================================================================

def plot_single_feature_comparison():
    fig, ax = plt.subplots(figsize=(10, 6))
    
    categories = ['Native Go\n(E2E Test)', 'WASM\n(Browser)']
    times = [native_time, wasm_time]
    colors_list = [colors['native'], colors['wasm']]
    
    bars = ax.bar(categories, times, color=colors_list, edgecolor='black', linewidth=1.5)
    
    # Add value labels on bars
    for bar, time in zip(bars, times):
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{time:.2f}ms',
                ha='center', va='bottom', fontsize=14, fontweight='bold')
    
    # Add slowdown factor
    ax.text(1, wasm_time + 10, f'526x slower!', 
            ha='center', fontsize=16, color='red', fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='yellow', alpha=0.7))
    
    ax.set_ylabel('Time (ms)', fontsize=14, fontweight='bold')
    ax.set_title('Single Feature Encryption Time\nNative Go vs WASM', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_ylim(0, wasm_time * 1.2)
    ax.grid(axis='y', alpha=0.3)
    
    plt.tight_layout()
    plt.savefig('performance_single_feature.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_single_feature.png")
    plt.close()

# ============================================================================
# Figure 2: 5 Features Comparison (Bar Chart)
# ============================================================================

def plot_5_features_comparison():
    fig, ax = plt.subplots(figsize=(10, 6))
    
    categories = ['Native Go\n(E2E Test)', 'WASM\n(Browser)']
    times = [native_5_features, wasm_5_features]
    colors_list = [colors['native'], colors['wasm']]
    
    bars = ax.bar(categories, times, color=colors_list, edgecolor='black', linewidth=1.5)
    
    # Add value labels on bars
    for bar, time in zip(bars, times):
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{time:.2f}ms',
                ha='center', va='bottom', fontsize=14, fontweight='bold')
    
    # Add slowdown factor
    ax.text(1, wasm_5_features + 30, f'600x slower!', 
            ha='center', fontsize=16, color='red', fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='yellow', alpha=0.7))
    
    ax.set_ylabel('Time (ms)', fontsize=14, fontweight='bold')
    ax.set_title('5 Features Encryption Time\nNative Go vs WASM', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_ylim(0, wasm_5_features * 1.2)
    ax.grid(axis='y', alpha=0.3)
    
    plt.tight_layout()
    plt.savefig('performance_5_features.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_5_features.png")
    plt.close()

# ============================================================================
# Figure 3: WASM Breakdown (Pie Chart)
# ============================================================================

def plot_wasm_breakdown():
    fig, ax = plt.subplots(figsize=(10, 8))
    
    labels = list(wasm_breakdown.keys())
    sizes = list(wasm_breakdown.values())
    colors_pie = ['#95a5a6', '#3498db', '#9b59b6', '#e74c3c', '#95a5a6', '#34495e']
    explode = (0, 0, 0, 0.1, 0, 0)  # Explode the bottleneck
    
    wedges, texts, autotexts = ax.pie(sizes, explode=explode, labels=labels, 
                                        colors=colors_pie, autopct='%1.1f%%',
                                        shadow=True, startangle=90,
                                        textprops={'fontsize': 12, 'fontweight': 'bold'})
    
    # Make percentage text more visible
    for autotext in autotexts:
        autotext.set_color('white')
        autotext.set_fontsize(11)
        autotext.set_fontweight('bold')
    
    ax.set_title('WASM Encryption Time Breakdown\n(Single Feature = 120ms)', 
                 fontsize=16, fontweight='bold', pad=20)
    
    # Add legend with actual times
    legend_labels = [f'{label}: {time:.1f}ms' for label, time in wasm_breakdown.items()]
    ax.legend(legend_labels, loc='center left', bbox_to_anchor=(1, 0, 0.5, 1), fontsize=10)
    
    # Highlight bottleneck
    ax.text(0, -1.4, '🔥 Bottleneck: Actual Encryption (92.8ms = 77%)', 
            ha='center', fontsize=14, color='red', fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='yellow', alpha=0.7))
    
    plt.tight_layout()
    plt.savefig('performance_wasm_breakdown.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_wasm_breakdown.png")
    plt.close()

# ============================================================================
# Figure 4: WASM Breakdown (Stacked Bar)
# ============================================================================

def plot_wasm_breakdown_stacked():
    fig, ax = plt.subplots(figsize=(12, 6))
    
    labels = list(wasm_breakdown.keys())
    sizes = list(wasm_breakdown.values())
    colors_stack = ['#95a5a6', '#3498db', '#9b59b6', '#e74c3c', '#95a5a6', '#34495e']
    
    # Create stacked bar
    bottom = 0
    bars = []
    for i, (label, size) in enumerate(zip(labels, sizes)):
        bar = ax.barh([0], [size], left=bottom, color=colors_stack[i], 
                      edgecolor='black', linewidth=1, label=label)
        bars.append(bar)
        
        # Add text label
        ax.text(bottom + size/2, 0, f'{size:.1f}ms\n{label}', 
                ha='center', va='center', fontsize=10, fontweight='bold',
                color='white' if i == 3 else 'black')
        
        bottom += size
    
    ax.set_xlim(0, sum(sizes) * 1.05)
    ax.set_ylim(-0.5, 0.5)
    ax.set_xlabel('Time (ms)', fontsize=14, fontweight='bold')
    ax.set_title('WASM Encryption Pipeline Breakdown (120ms Total)', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_yticks([])
    ax.legend(loc='upper center', bbox_to_anchor=(0.5, -0.05), ncol=3, fontsize=10)
    ax.grid(axis='x', alpha=0.3)
    
    # Add total time
    ax.text(sum(sizes) + 5, 0, f'Total: {sum(sizes):.1f}ms', 
            ha='left', va='center', fontsize=12, fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='lightblue', alpha=0.7))
    
    plt.tight_layout()
    plt.savefig('performance_wasm_pipeline.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_wasm_pipeline.png")
    plt.close()

# ============================================================================
# Figure 5: E2E Comparison (Stacked Bar)
# ============================================================================

def plot_e2e_comparison():
    fig, ax = plt.subplots(figsize=(12, 7))
    
    # Prepare data
    stages = list(e2e_native.keys())
    native_values = list(e2e_native.values())
    wasm_values = list(e2e_wasm.values())
    
    x = np.arange(len(stages))
    width = 0.35
    
    # Create grouped bars
    bars1 = ax.bar(x - width/2, native_values, width, label='Native Go', 
                   color=colors['native'], edgecolor='black', linewidth=1.5)
    bars2 = ax.bar(x + width/2, wasm_values, width, label='WASM', 
                   color=colors['wasm'], edgecolor='black', linewidth=1.5)
    
    # Add value labels
    for bars in [bars1, bars2]:
        for bar in bars:
            height = bar.get_height()
            ax.text(bar.get_x() + bar.get_width()/2., height,
                    f'{height:.1f}ms' if height < 10 else f'{height:.0f}ms',
                    ha='center', va='bottom', fontsize=10, fontweight='bold')
    
    ax.set_xlabel('Pipeline Stage', fontsize=14, fontweight='bold')
    ax.set_ylabel('Time (ms)', fontsize=14, fontweight='bold')
    ax.set_title('End-to-End Pipeline Comparison\nNative Go vs WASM', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_xticks(x)
    ax.set_xticklabels(stages, fontsize=11)
    ax.legend(fontsize=12)
    ax.grid(axis='y', alpha=0.3)
    
    # Add total time annotations
    total_native = sum(native_values)
    total_wasm = sum(wasm_values)
    ax.text(len(stages) - 0.5, max(wasm_values) * 1.1, 
            f'Native Total: {total_native:.1f}ms\nWASM Total: {total_wasm:.0f}ms\n({total_wasm/total_native:.1f}x slower)', 
            ha='center', fontsize=12, fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='lightyellow', alpha=0.8))
    
    plt.tight_layout()
    plt.savefig('performance_e2e_comparison.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_e2e_comparison.png")
    plt.close()

# ============================================================================
# Figure 6: Speedup Factor (Horizontal Bar)
# ============================================================================

def plot_speedup_factors():
    fig, ax = plt.subplots(figsize=(10, 6))
    
    operations = ['Single Feature', '5 Features', 'Key Generation', 'Decryption']
    speedup_factors = [
        wasm_time / native_time,  # 526x
        wasm_5_features / native_5_features,  # 600x
        e2e_wasm['Key Gen'] / e2e_native['Key Gen'],  # 16x
        e2e_wasm['Decrypt'] / e2e_native['Decrypt']  # 15x
    ]
    
    colors_list = [colors['wasm'] if x > 100 else colors['encoding'] for x in speedup_factors]
    
    bars = ax.barh(operations, speedup_factors, color=colors_list, 
                   edgecolor='black', linewidth=1.5)
    
    # Add value labels
    for bar, factor in zip(bars, speedup_factors):
        width = bar.get_width()
        ax.text(width + 10, bar.get_y() + bar.get_height()/2.,
                f'{factor:.0f}x slower',
                ha='left', va='center', fontsize=12, fontweight='bold')
    
    ax.set_xlabel('Slowdown Factor (WASM vs Native)', fontsize=14, fontweight='bold')
    ax.set_title('WASM Performance Penalty Across Operations', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_xlim(0, max(speedup_factors) * 1.2)
    ax.grid(axis='x', alpha=0.3)
    ax.axvline(x=100, color='red', linestyle='--', linewidth=2, alpha=0.5, label='100x threshold')
    ax.legend(fontsize=10)
    
    plt.tight_layout()
    plt.savefig('performance_speedup_factors.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_speedup_factors.png")
    plt.close()

# ============================================================================
# Figure 7: Comprehensive Dashboard
# ============================================================================

def plot_comprehensive_dashboard():
    fig = plt.figure(figsize=(16, 10))
    gs = GridSpec(3, 3, figure=fig, hspace=0.3, wspace=0.3)
    
    # 1. Single feature comparison
    ax1 = fig.add_subplot(gs[0, 0])
    categories = ['Native', 'WASM']
    times = [native_time, wasm_time]
    bars = ax1.bar(categories, times, color=[colors['native'], colors['wasm']], 
                   edgecolor='black', linewidth=1.5)
    for bar, time in zip(bars, times):
        height = bar.get_height()
        ax1.text(bar.get_x() + bar.get_width()/2., height,
                f'{time:.2f}ms', ha='center', va='bottom', fontsize=10, fontweight='bold')
    ax1.set_ylabel('Time (ms)', fontsize=10, fontweight='bold')
    ax1.set_title('Single Feature', fontsize=12, fontweight='bold')
    ax1.grid(axis='y', alpha=0.3)
    
    # 2. 5 features comparison
    ax2 = fig.add_subplot(gs[0, 1])
    times_5 = [native_5_features, wasm_5_features]
    bars = ax2.bar(categories, times_5, color=[colors['native'], colors['wasm']], 
                   edgecolor='black', linewidth=1.5)
    for bar, time in zip(bars, times_5):
        height = bar.get_height()
        ax2.text(bar.get_x() + bar.get_width()/2., height,
                f'{time:.1f}ms', ha='center', va='bottom', fontsize=10, fontweight='bold')
    ax2.set_ylabel('Time (ms)', fontsize=10, fontweight='bold')
    ax2.set_title('5 Features', fontsize=12, fontweight='bold')
    ax2.grid(axis='y', alpha=0.3)
    
    # 3. Speedup factors
    ax3 = fig.add_subplot(gs[0, 2])
    ops = ['1 Feat', '5 Feat', 'KeyGen', 'Decrypt']
    factors = [526, 600, 16, 15]
    bars = ax3.barh(ops, factors, color=[colors['wasm'], colors['wasm'], 
                                          colors['encoding'], colors['encoding']], 
                    edgecolor='black', linewidth=1.5)
    for bar, factor in zip(bars, factors):
        width = bar.get_width()
        ax3.text(width + 10, bar.get_y() + bar.get_height()/2.,
                f'{factor}x', ha='left', va='center', fontsize=9, fontweight='bold')
    ax3.set_xlabel('Slowdown', fontsize=10, fontweight='bold')
    ax3.set_title('WASM Penalty', fontsize=12, fontweight='bold')
    ax3.grid(axis='x', alpha=0.3)
    
    # 4. WASM breakdown pie
    ax4 = fig.add_subplot(gs[1, :2])
    labels = list(wasm_breakdown.keys())
    sizes = list(wasm_breakdown.values())
    colors_pie = ['#95a5a6', '#3498db', '#9b59b6', '#e74c3c', '#95a5a6', '#34495e']
    explode = (0, 0, 0, 0.05, 0, 0)
    wedges, texts, autotexts = ax4.pie(sizes, explode=explode, labels=labels, 
                                         colors=colors_pie, autopct='%1.1f%%',
                                         startangle=90, textprops={'fontsize': 9})
    for autotext in autotexts:
        autotext.set_color('white')
        autotext.set_fontweight('bold')
    ax4.set_title('WASM Breakdown (120ms)', fontsize=12, fontweight='bold')
    
    # 5. WASM pipeline stacked
    ax5 = fig.add_subplot(gs[1, 2])
    labels_short = ['PK', 'Enc', 'Encr\nCrt', 'Encrypt', 'CT', 'JS']
    sizes = list(wasm_breakdown.values())
    colors_stack = ['#95a5a6', '#3498db', '#9b59b6', '#e74c3c', '#95a5a6', '#34495e']
    bottom = 0
    for i, (label, size, color) in enumerate(zip(labels_short, sizes, colors_stack)):
        ax5.bar([0], [size], bottom=bottom, color=color, edgecolor='black', linewidth=1)
        if size > 5:
            ax5.text(0, bottom + size/2, f'{size:.1f}', ha='center', va='center', 
                    fontsize=8, fontweight='bold', color='white' if i == 3 else 'black')
        bottom += size
    ax5.set_xlim(-0.5, 0.5)
    ax5.set_ylabel('Time (ms)', fontsize=10, fontweight='bold')
    ax5.set_title('Pipeline Stack', fontsize=12, fontweight='bold')
    ax5.set_xticks([])
    ax5.grid(axis='y', alpha=0.3)
    
    # 6. E2E comparison
    ax6 = fig.add_subplot(gs[2, :])
    stages = list(e2e_native.keys())
    native_values = list(e2e_native.values())
    wasm_values = list(e2e_wasm.values())
    x = np.arange(len(stages))
    width = 0.35
    bars1 = ax6.bar(x - width/2, native_values, width, label='Native Go', 
                    color=colors['native'], edgecolor='black', linewidth=1.5)
    bars2 = ax6.bar(x + width/2, wasm_values, width, label='WASM', 
                    color=colors['wasm'], edgecolor='black', linewidth=1.5)
    for bars in [bars1, bars2]:
        for bar in bars:
            height = bar.get_height()
            ax6.text(bar.get_x() + bar.get_width()/2., height,
                    f'{height:.0f}' if height >= 10 else f'{height:.1f}',
                    ha='center', va='bottom', fontsize=9, fontweight='bold')
    ax6.set_xlabel('Stage', fontsize=10, fontweight='bold')
    ax6.set_ylabel('Time (ms)', fontsize=10, fontweight='bold')
    ax6.set_title('E2E Pipeline: Native (480ms) vs WASM (1492ms)', 
                  fontsize=12, fontweight='bold')
    ax6.set_xticks(x)
    ax6.set_xticklabels(stages, fontsize=9)
    ax6.legend(fontsize=10)
    ax6.grid(axis='y', alpha=0.3)
    
    # Add main title
    fig.suptitle('CKKS Performance Analysis: Native Go vs WASM\n' + 
                 '🚀 Native: 0.19ms/feature | 🐌 WASM: 100ms/feature (526x slower)', 
                 fontsize=16, fontweight='bold', y=0.98)
    
    plt.savefig('performance_dashboard.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_dashboard.png")
    plt.close()

# ============================================================================
# Figure 8: Optimization Impact
# ============================================================================

def plot_optimization_impact():
    fig, ax = plt.subplots(figsize=(12, 7))
    
    optimizations = ['Before\nOptimization', 'Encoder\nCaching', 'Build\nFlags', 
                     'Memory\nPooling\n(theoretical)', 'Final\nResult']
    times = [1500, 582, 582, 550, 582]  # ms for 5 features
    colors_opt = ['#e74c3c', '#e67e22', '#f39c12', '#f1c40f', '#e74c3c']
    
    bars = ax.bar(optimizations, times, color=colors_opt, edgecolor='black', linewidth=1.5)
    
    # Add value labels and improvement percentages
    for i, (bar, time) in enumerate(zip(bars, times)):
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width()/2., height,
                f'{time}ms',
                ha='center', va='bottom', fontsize=12, fontweight='bold')
        
        if i > 0:
            improvement = ((times[0] - time) / times[0]) * 100
            ax.text(bar.get_x() + bar.get_width()/2., height * 0.5,
                    f'-{improvement:.0f}%' if improvement > 0 else 'No change',
                    ha='center', va='center', fontsize=10, fontweight='bold',
                    color='white', bbox=dict(boxstyle='round', facecolor='green' if improvement > 0 else 'gray', alpha=0.7))
    
    # Add baseline comparison
    ax.axhline(y=native_5_features, color='green', linestyle='--', linewidth=2, 
               label=f'Native Go Target ({native_5_features:.1f}ms)', alpha=0.7)
    
    ax.set_ylabel('Time (ms)', fontsize=14, fontweight='bold')
    ax.set_title('Optimization Attempts Impact (5 Features)\nWASM Performance Improvements', 
                 fontsize=16, fontweight='bold', pad=20)
    ax.set_ylim(0, 1600)
    ax.legend(fontsize=12, loc='upper right')
    ax.grid(axis='y', alpha=0.3)
    
    # Add reality check
    ax.text(2, 1400, 'Reality Check:\nWASM cannot reach\nNative performance!', 
            ha='center', fontsize=12, color='red', fontweight='bold',
            bbox=dict(boxstyle='round', facecolor='yellow', alpha=0.7))
    
    plt.tight_layout()
    plt.savefig('performance_optimization_impact.png', dpi=300, bbox_inches='tight')
    print("✅ Saved: performance_optimization_impact.png")
    plt.close()

# ============================================================================
# Main execution
# ============================================================================

if __name__ == '__main__':
    print("\n" + "="*70)
    print("🔬 CKKS Performance Visualization Generator")
    print("="*70 + "\n")
    
    print("📊 Generating performance charts...\n")
    
    try:
        plot_single_feature_comparison()
        plot_5_features_comparison()
        plot_wasm_breakdown()
        plot_wasm_breakdown_stacked()
        plot_e2e_comparison()
        plot_speedup_factors()
        plot_optimization_impact()
        plot_comprehensive_dashboard()
        
        print("\n" + "="*70)
        print("✅ All charts generated successfully!")
        print("="*70)
        print("\nGenerated files:")
        print("  1. performance_single_feature.png     - Single feature comparison")
        print("  2. performance_5_features.png         - 5 features comparison")
        print("  3. performance_wasm_breakdown.png     - WASM breakdown pie chart")
        print("  4. performance_wasm_pipeline.png      - WASM pipeline stacked bar")
        print("  5. performance_e2e_comparison.png     - E2E pipeline comparison")
        print("  6. performance_speedup_factors.png    - Slowdown factors")
        print("  7. performance_optimization_impact.png - Optimization attempts")
        print("  8. performance_dashboard.png          - Comprehensive dashboard")
        print("\n" + "="*70)
        
    except Exception as e:
        print(f"\n❌ Error generating charts: {e}")
        import traceback
        traceback.print_exc()
