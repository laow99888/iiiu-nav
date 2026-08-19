import { render, screen } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { ChevronDown, Search } from 'lucide-preact';
import { describe, expect, it, vi } from 'vitest';

import { Button } from './button';
import { EmptyState } from './empty-state';
import { TextField } from './field';
import { FileInput } from './file-input';

describe('基础控件', () => {
  it('按钮覆盖图标、加载、禁用和焦点状态', async () => {
    const user = userEvent.setup();
    render(
      <div>
        <Button icon={Search} aria-label="搜索" />
        <Button loading>保存</Button>
        <Button disabled>删除</Button>
      </div>,
    );

    const iconButton = screen.getByRole('button', { name: '搜索' });
    await user.tab();
    expect(iconButton).toHaveFocus();
    expect(screen.getByRole('button', { name: '保存' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '保存' })).toHaveAttribute(
      'aria-busy',
      'true',
    );
    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled();
  });

  it('按钮文字和内嵌符号共享内容对齐盒', () => {
    render(
      <Button>
        Google
        <ChevronDown aria-hidden="true" />
      </Button>,
    );

    const content = screen.getByText('Google');
    expect(content).toHaveClass('ui-button__content');
    expect(content.querySelector('svg')).toBeInTheDocument();
  });

  it('字段关联说明和错误，并保留禁用状态', async () => {
    const user = userEvent.setup();
    render(
      <div>
        <TextField
          label="名称"
          description="显示在卡片上"
          error="名称不能为空"
        />
        <TextField label="只读字段" disabled value="固定值" />
      </div>,
    );

    const input = screen.getByRole('textbox', { name: '名称' });
    await user.click(input);
    expect(input).toHaveFocus();
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(input).toHaveAccessibleDescription('显示在卡片上 名称不能为空');
    expect(screen.getByRole('textbox', { name: '只读字段' })).toBeDisabled();
  });

  it('文件输入覆盖选择、错误、加载和禁用状态', async () => {
    const user = userEvent.setup();
    const onFilesChange = vi.fn();
    const { rerender } = render(
      <FileInput
        label="上传图标"
        error="只支持图片"
        onFilesChange={onFilesChange}
      />,
    );
    const input = screen.getByLabelText('上传图标');
    const file = new File(['icon'], 'logo.png', { type: 'image/png' });

    input.focus();
    expect(input).toHaveFocus();
    await user.upload(input, file);
    expect(onFilesChange).toHaveBeenCalledWith([file]);
    expect(screen.getByText('logo.png')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('只支持图片');

    rerender(
      <FileInput label="正在上传" loading onFilesChange={onFilesChange} />,
    );
    expect(screen.getByLabelText('正在上传')).toBeDisabled();
    rerender(
      <FileInput label="禁止上传" disabled onFilesChange={onFilesChange} />,
    );
    expect(screen.getByLabelText('禁止上传')).toBeDisabled();
  });

  it('空状态覆盖默认、加载和错误呈现', () => {
    const { rerender } = render(
      <EmptyState
        title="暂无内容"
        description="添加第一条链接"
        icon={Search}
      />,
    );
    expect(screen.getByText('暂无内容')).toBeInTheDocument();

    rerender(
      <EmptyState title="正在加载" description="请稍候" state="loading" />,
    );
    expect(screen.getByText('正在加载').parentElement).toHaveAttribute(
      'aria-busy',
      'true',
    );

    rerender(
      <EmptyState title="加载失败" description="请重试" state="error" />,
    );
    expect(screen.getByText('加载失败').parentElement).toHaveClass(
      'ui-empty--error',
    );
  });
});
