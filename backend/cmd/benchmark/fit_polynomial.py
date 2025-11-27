import numpy as np
from scipy.optimize import curve_fit
import matplotlib.pyplot as plt

# Sigmoid function
def sigmoid(x):
    return 1.0 / (1.0 + np.exp(-x))

# Polynomial model
def polynomial(x, *coeffs):
    result = np.zeros_like(x)
    for i, c in enumerate(coeffs):
        result += c * (x ** i)
    return result

# Credit scoring range
a, b = -3.0, -1.0

# Generate dense sample points
n_points = 1000
x_data = np.linspace(a, b, n_points)
y_data = sigmoid(x_data)

# Fit polynomials of different degrees
degrees = [3, 5, 7]

print("Polynomial Fit Coefficients for Sigmoid in [-3, -1]")
print("=" * 60)

for degree in degrees:
    # Fit polynomial
    p0 = np.zeros(degree + 1)  # Initial guess
    popt, _ = curve_fit(polynomial, x_data, y_data, p0=p0)
    
    print(f"\nDegree {degree}:")
    print("-" * 40)
    print("Coefficients:")
    for i, c in enumerate(popt):
        print(f"  c[{i}] = {c:12.8f}")
    
    # Test on key points
    test_points = [-3.0, -2.5, -2.0, -1.5, -1.0]
    print("\nTest Points:")
    
    max_error = 0
    total_error = 0
    
    for x in test_points:
        expected = sigmoid(x)
        approx = polynomial(x, *popt)
        error = abs(expected - approx)
        rel_error = error / expected * 100
        
        print(f"  x={x:4.1f}: expected={expected:.6f}, approx={approx:.6f}, "
              f"error={error:.6f} ({rel_error:5.2f}%)")
        
        max_error = max(max_error, error)
        total_error += error
    
    avg_error = total_error / len(test_points)
    print(f"\nMax Error: {max_error:.6f} ({max_error/0.15*100:.2f}% of range)")
    print(f"Avg Error: {avg_error:.6f} ({avg_error/0.15*100:.2f}% of range)")
    
    # Verify over entire range
    y_approx = polynomial(x_data, *popt)
    errors = np.abs(y_data - y_approx)
    print(f"\nRange-wide statistics:")
    print(f"  Max error: {np.max(errors):.8f}")
    print(f"  Mean error: {np.mean(errors):.8f}")
    print(f"  RMS error: {np.sqrt(np.mean(errors**2)):.8f}")

# Generate Go code
print("\n" + "=" * 60)
print("GO CODE GENERATION:")
print("=" * 60)
for degree in degrees:
    p0 = np.zeros(degree + 1)
    popt, _ = curve_fit(polynomial, x_data, y_data, p0=p0)
    
    print(f"\ncase {degree}:")
    print(f"\t// {degree}차 다항식 (degree {degree} polynomial fit)")
    print(f"\tcoeffs = []float64{{")
    for i, c in enumerate(popt):
        print(f"\t\t{c:12.8f},   // c{i}: x^{i} term")
    print(f"\t}}")
